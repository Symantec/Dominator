package filesystem

import (
	"sync"
	"syscall"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver/filesystem/scan"
)

func newObjectServer(baseDir string, logger log.Logger) (
	*ObjectServer, error) {
	startTime := time.Now()
	var rusageStart, rusageStop syscall.Rusage
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStart)
	sizesMap := make(map[hash.Hash]uint64)
	var mutex sync.Mutex
	err := scan.ScanTree(baseDir, func(hashVal hash.Hash, size uint64) {
		mutex.Lock()
		sizesMap[hashVal] = size
		mutex.Unlock()
	})
	if err != nil {
		return nil, err
	}
	plural := ""
	if len(sizesMap) != 1 {
		plural = "s"
	}
	syscall.Getrusage(syscall.RUSAGE_SELF, &rusageStop)
	userTime := time.Duration(rusageStop.Utime.Sec)*time.Second +
		time.Duration(rusageStop.Utime.Usec)*time.Microsecond -
		time.Duration(rusageStart.Utime.Sec)*time.Second -
		time.Duration(rusageStart.Utime.Usec)*time.Microsecond
	logger.Printf("Scanned %d object%s in %s (%s user CPUtime)\n",
		len(sizesMap), plural, time.Since(startTime), userTime)
	return &ObjectServer{
		baseDir:               baseDir,
		logger:                logger,
		sizesMap:              sizesMap,
		lastGarbageCollection: time.Now(),
		lastMutationTime:      time.Now(),
	}, nil
}
