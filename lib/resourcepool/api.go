package resourcepool

import (
	"errors"
	"sync"
)

var (
	ErrorResourceLimitExceeded = errors.New("resource limit exceeded")
)

type AllocateReleaser interface {
	Allocate() error
	Release() error
}

type Pool struct {
	max       uint
	semaphore chan struct{}
	lock      sync.Mutex
	numUsed   uint
	unused    map[*Resource]struct{}
}

type Resource struct {
	pool             *Pool
	allocateReleaser AllocateReleaser
	allocating       bool
	inUse            bool
	releaseOnPut     bool
	allocated        bool
	releaseError     error
}

func New(max uint) *Pool {
	return &Pool{
		max:       max,
		semaphore: make(chan struct{}, max),
		unused:    make(map[*Resource]struct{}),
	}
}

func (pool *Pool) Create(allocateReleaser AllocateReleaser) *Resource {
	return &Resource{pool: pool, allocateReleaser: allocateReleaser}
}

func (resource *Resource) Get(cancelChannel <-chan struct{}) error {
	return resource.get(cancelChannel)
}

func (resource *Resource) Put() {
	resource.put()
}

func (resource *Resource) Release() error {
	return resource.release(false)
}

func (resource *Resource) ScheduleRelease() error {
	return resource.scheduleRelease()
}
