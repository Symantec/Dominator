package connpool

import (
	"sync"
	"syscall"

	"github.com/Symantec/Dominator/lib/resourcepool"
)

var (
	lock sync.Mutex
	pool *resourcepool.Pool
)

func getConnectionLimit() uint {
	var rlim syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim); err != nil {
		return 900
	}
	maxConnAttempts := rlim.Cur - 50
	maxConnAttempts = (maxConnAttempts / 100)
	if maxConnAttempts < 1 {
		maxConnAttempts = 1
	} else {
		maxConnAttempts *= 100
	}
	return uint(maxConnAttempts)
}

func getResourcePool() *resourcepool.Pool {
	// Delay setting of internal limits to allow application code to increase
	// the limit on file descriptors first.
	if pool == nil {
		lock.Lock()
		if pool == nil {
			pool = resourcepool.New(getConnectionLimit(), "connections")
		}
		lock.Unlock()
	}
	return pool
}
