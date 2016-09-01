package resourcepool

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
	pool.lock.Lock()
	defer pool.lock.Unlock()
	if resource.allocated {
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
		if !resourceToRelease.allocated {
			panic("Resource is not allocated")
		}
		delete(pool.unused, resourceToRelease)
		if resourceToRelease.releaseFunc != nil {
			resourceToRelease.releaseFunc()
			resourceToRelease.releaseFunc = nil
		}
		resourceToRelease.allocated = false
	}
	resource.inUse = true
	resource.releaseFunc = nil
	resource.allocated = true
	pool.numUsed++
	return true
}

func (resource *Resource) put() {
	pool := resource.pool
	pool.lock.Lock()
	if !resource.allocated {
		pool.lock.Unlock()
		return
	}
	if !resource.inUse {
		pool.lock.Unlock()
		panic("Resource was not gotten")
	}
	resource.inUse = false
	if resource.releaseOnPut {
		if resource.releaseFunc != nil {
			resource.releaseFunc()
			resource.releaseFunc = nil
		}
		resource.allocated = false
	} else if resource.releaseFunc != nil {
		pool.unused[resource] = struct{}{}
	}
	pool.numUsed--
	pool.lock.Unlock()
	<-pool.semaphore // Free up a slot for someone else.
}

func (resource *Resource) release(haveLock bool) {
	pool := resource.pool
	if !haveLock {
		pool.lock.Lock()
	}
	if !resource.allocated {
		pool.lock.Unlock()
		return
	}
	if resource.releaseFunc != nil {
		resource.releaseFunc()
		resource.releaseFunc = nil
	}
	resource.allocated = false
	delete(resource.pool.unused, resource)
	wasUsed := resource.inUse
	if resource.inUse {
		resource.inUse = false
		pool.numUsed--
	}
	pool.lock.Unlock()
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
	if resource.releaseFunc != nil {
		panic("Cannot change releaseFunc once set")
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
