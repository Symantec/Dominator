package resourcepool

func (pool *Pool) getSlot(cancelChannel <-chan struct{}) bool {
	// Grab a slot (the right to have a resource in use).
	select {
	case pool.semaphore <- struct{}{}:
		return true
	default:
	}
	select {
	case pool.semaphore <- struct{}{}:
		return true
	case <-cancelChannel:
		return false
	}
}

func (resource *Resource) get(cancelChannel <-chan struct{}) error {
	pool := resource.pool
	if resource.inUse {
		panic("Resource is already in use")
	}
	if !pool.getSlot(cancelChannel) {
		return ErrorResourceLimitExceeded
	}
	pool.lock.Lock()
	defer pool.lock.Unlock()
	if resource.allocated {
		delete(pool.unused, resource)
		resource.inUse = true
		pool.numUsed++
		return nil
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
		resourceToRelease.releaseError =
			resourceToRelease.allocateReleaser.Release()
		resourceToRelease.allocated = false
	}
	resource.allocating = true
	resource.inUse = true
	resource.allocated = true
	pool.numUsed++
	pool.lock.Unlock()
	err := resource.allocateReleaser.Allocate()
	pool.lock.Lock()
	resource.allocating = false
	if err != nil {
		resource.inUse = false
		resource.allocated = false
		pool.numUsed--
		<-pool.semaphore // Free up a slot for someone else.
		return err
	}
	return nil
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
		resource.releaseError = resource.allocateReleaser.Release()
		resource.allocated = false
	} else {
		pool.unused[resource] = struct{}{}
	}
	pool.numUsed--
	pool.lock.Unlock()
	<-pool.semaphore // Free up a slot for someone else.
}

func (resource *Resource) release(haveLock bool) error {
	pool := resource.pool
	if !haveLock {
		pool.lock.Lock()
	}
	if resource.allocating {
		pool.lock.Unlock()
		panic("Resource is allocating")
	}
	if !resource.allocated {
		pool.lock.Unlock()
		return resource.releaseError
	}
	resource.releaseError = resource.allocateReleaser.Release()
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
	return resource.releaseError
}

func (resource *Resource) scheduleRelease() error {
	resource.pool.lock.Lock()
	if resource.inUse {
		resource.releaseOnPut = true
		resource.pool.lock.Unlock()
		return nil
	}
	return resource.release(true)
}
