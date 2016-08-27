package resourcepool

import ()

func (pool *Pool) getSlot(wait bool) bool {
	// Grab a slot (the right to have a resource in use).
	if wait {
		pool.semaphore <- struct{}{}
	} else {
		select {
		case pool.semaphore <- struct{}{}:
		default:
			return false
		}
	}
	return true
}

func (resource *Resource) get(wait bool) bool {
	pool := resource.pool
	if resource.inUse {
		panic("Resource is already in use")
	}
	if !pool.getSlot(wait) {
		return false
	}
	var releaseFunc func()
	defer func() {
		if releaseFunc != nil {
			releaseFunc()
		}
	}()
	pool.lock.Lock()
	defer pool.lock.Unlock()
	if resource.releaseFunc != nil {
		delete(pool.unused, resource)
		resource.inUse = true
		pool.numUsed++
		return true
	}
	if pool.numUsed+uint(len(pool.unused)) >= pool.max {
		// Need to grab a free resource and release. Be lazy: do a random pick.
		var resourceToRelease *Resource
		for res := range pool.unused {
			resourceToRelease = res
			break
		}
		if resourceToRelease == nil {
			panic("No free resource to release")
		}
		delete(pool.unused, resourceToRelease)
		releaseFunc = resourceToRelease.releaseFunc
		resourceToRelease.releaseFunc = nil
	}
	resource.inUse = true
	resource.releaseFunc = nil
	pool.numUsed++
	return true
}

func (resource *Resource) put() {
	pool := resource.pool
	pool.lock.Lock()
	if resource.released {
		pool.lock.Unlock()
		return
	}
	if !resource.inUse {
		pool.lock.Unlock()
		panic("Resource was not gotten")
	}
	resource.inUse = false
	var releaseFunc func()
	if resource.releaseOnPut {
		releaseFunc = resource.releaseFunc
	} else if resource.releaseFunc != nil {
		pool.unused[resource] = struct{}{}
	}
	pool.numUsed--
	pool.lock.Unlock()
	if releaseFunc != nil {
		releaseFunc()
	}
	<-pool.semaphore // Free up a slot for someone else.
}

func (resource *Resource) release(haveLock bool) {
	pool := resource.pool
	if !haveLock {
		pool.lock.Lock()
	}
	if resource.released {
		pool.lock.Unlock()
		return
	}
	resource.released = true
	releaseFunc := resource.releaseFunc
	resource.releaseFunc = nil
	delete(resource.pool.unused, resource)
	wasUsed := resource.inUse
	if resource.inUse {
		resource.inUse = false
		pool.numUsed--
	}
	pool.lock.Unlock()
	if releaseFunc != nil {
		releaseFunc()
	}
	if wasUsed {
		<-pool.semaphore // Free up a slot for someone else.
	}
}

func (resource *Resource) setReleaseFunc(releaseFunc func()) {
	if releaseFunc == nil {
		panic("Cannot set nil releaseFunc")
	}
	resource.pool.lock.Lock()
	defer resource.pool.lock.Unlock()
	if !resource.inUse {
		panic("Resource was not gotten")
	}
	resource.releaseFunc = releaseFunc
}

func (resource *Resource) scheduleRelease() {
	resource.pool.lock.Lock()
	if resource.inUse {
		resource.releaseOnPut = true
		resource.pool.lock.Unlock()
		return
	}
	resource.release(true)
}
