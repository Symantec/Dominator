package cachingreader

import (
	"sync"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver/filesystem/scan"
)

func newObjectServer(baseDir string, maxCachedBytes uint64,
	objectServerAddress string, logger log.DebugLogger) (*ObjectServer, error) {
	startTime := time.Now()
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	var mutex sync.Mutex
	objects := make(map[hash.Hash]*objectType)
	var cachedBytes uint64
	err := scan.ScanTree(baseDir, func(hashVal hash.Hash, size uint64) {
		mutex.Lock()
		cachedBytes += size
		objects[hashVal] = &objectType{hash: hashVal, size: size}
		mutex.Unlock()
	})
	if err != nil {
		return nil, err
	}
	plural := ""
	if len(objects) != 1 {
		plural = "s"
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	userTime := time.Duration(rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	logger.Printf("Scanned %d object%s in %s (%s user CPUtime)\n",
		len(objects), plural, time.Since(startTime), userTime)
	lruUpdateNotifier := make(chan struct{}, 1)
	objSrv := &ObjectServer{
		baseDir:             baseDir,
		flushTimer:          time.NewTimer(time.Minute),
		logger:              logger,
		lruUpdateNotifier:   lruUpdateNotifier,
		maxCachedBytes:      maxCachedBytes,
		objectServerAddress: objectServerAddress,
		cachedBytes:         cachedBytes,
		objects:             objects,
	}
	objSrv.flushTimer.Stop()
	if err := objSrv.loadLru(); err != nil {
		return nil, err
	}
	// Link orphaned entries.
	for _, object := range objSrv.objects {
		if objSrv.newest == nil { // Empty list: initialise it.
			objSrv.newest = object
			objSrv.oldest = object
			objSrv.lruBytes += object.size
		} else if object.newer == nil && objSrv.newest != object {
			// Orphaned object: make it the newest.
			object.older = objSrv.newest
			objSrv.newest.newer = object
			objSrv.newest = object
			objSrv.lruBytes += object.size
		}
	}
	go objSrv.flusher(lruUpdateNotifier)
	return objSrv, nil
}
