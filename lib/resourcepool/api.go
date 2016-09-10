/*
	Package resourcepool provides for managing shared resource pools.

	Package resourcepool may be used by other packages to manage shared
	resources (e.g. the lib/connpool package uses this package to help manage
	connection pools).
	A typical programming pattern is:
		pool := New(...)
		cr0 := pool.Create(...)
		cr1 := pool.Create(...)
		go func() {
			for ... {
				c := cr0.Get(...)
				defer c.Put()
				if err { c.Release() }
			}
		}()
		go func() {
			for ... {
				c := cr1.Get(...)
				defer c.Put()
				if err { c.Release() }
			}
		}()
	This pattern ensures Get and Put are always matched, and if there is an
	error, Release releases the underlying resource so that a subsequent Get
	creates a new underlying resource.

	It is resonable to create one goroutine for each resource, since the Get
	methods will block, waiting for available resources.
*/
package resourcepool

import (
	"errors"
	"sync"
)

var (
	ErrorResourceLimitExceeded = errors.New("resource limit exceeded")
)

// MakeImmediateCanceler returns a channel that may be passed to the
// Resource.Get method for callers that do not want to wait for a resource.
func MakeImmediateCanceler() <-chan struct{} {
	ch := make(chan struct{}, 1)
	ch <- struct{}{}
	return ch
}

// AllocateReleaser defines a type that can be used to allocate and release
// resources.
type AllocateReleaser interface {
	Allocate() error
	Release() error
}

// Pool groups and manages a set of resources.
type Pool struct {
	max          uint
	semaphore    chan struct{}
	lock         sync.Mutex
	numUsed      uint
	unused       map[*Resource]struct{}
	numReleasing uint
}

// Resource is a container for an underlying resource.
type Resource struct {
	pool             *Pool
	allocateReleaser AllocateReleaser
	allocating       bool
	inUse            bool
	releasing        sync.Mutex
	releaseOnPut     bool
	allocated        bool
	releaseError     error
}

// New returns a new resource Pool. The maximum number of resources that can be
// allocated concurrently is specified by max.
func New(max uint, metricsSubDirname string) *Pool {
	return &Pool{
		max:       max,
		semaphore: make(chan struct{}, max),
		unused:    make(map[*Resource]struct{}),
	}
}

// Create returns a new Resource for the pool. An unlimted number of resources
// may be created. The mechanism to specify how to allocate and release
// underlying resources is given by allocateReleaser. Create does not allocate
// resources.
func (pool *Pool) Create(allocateReleaser AllocateReleaser) *Resource {
	return &Resource{pool: pool, allocateReleaser: allocateReleaser}
}

// Get attempts to allocate an underlying resource. It calls the Allocate method
// of the AllocateReleaser passed to the Create method. Get will wait until a
// resource is available or a message is received on cancelChannel. If
// cancelChannel is nil then Get will wait indefinitely until a resource is
// available. If the wait is cancelled then Get will return
// ErrorResourceLimitExceeded. The resource is considered in use until a later
// call to Put or Release.
func (resource *Resource) Get(cancelChannel <-chan struct{}) error {
	return resource.get(cancelChannel)
}

// Put will free the resource, indicating that it is not currently needed. It
// may be internally released later if required to free limited resources. If
// Put is called after Release, no action is taken (this is a safe operation and
// is commonly used in some programming patterns).
func (resource *Resource) Put() {
	resource.put()
}

// Release will release the resource, immediately calling the Release method of
// the AllocateReleaser and returning its return value.
func (resource *Resource) Release() error {
	return resource.release(false)
}

// ScheduleRelease will immediately release the resource if it is not currently
// in use, otherwise it will schedule the resource to be released after the next
// Put.
func (resource *Resource) ScheduleRelease() error {
	return resource.scheduleRelease()
}
