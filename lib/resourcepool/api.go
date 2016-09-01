package resourcepool

import (
	"sync"
)

type Pool struct {
	max       uint
	semaphore chan struct{}
	lock      sync.Mutex
	numUsed   uint
	unused    map[*Resource]struct{}
}

type Resource struct {
	pool         *Pool
	inUse        bool
	releaseFunc  func()
	releaseOnPut bool
	allocated    bool
}

func New(max uint) *Pool {
	return &Pool{
		max:       max,
		semaphore: make(chan struct{}, max),
		unused:    make(map[*Resource]struct{}),
	}
}

func (pool *Pool) Create() *Resource {
	return &Resource{pool: pool}
}

func (resource *Resource) Get(wait bool) bool {
	return resource.get(wait)
}

func (resource *Resource) Put() {
	resource.put()
}

func (resource *Resource) Release() {
	resource.release(false)
}

func (resource *Resource) SetReleaseFunc(releaseFunc func()) {
	resource.setReleaseFunc(releaseFunc)
}

func (resource *Resource) ScheduleRelease() {
	resource.scheduleRelease()
}
