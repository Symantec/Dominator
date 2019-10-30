package scanner

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
)

func (fsh *FileSystemHistory) update(newFS *FileSystem) {
	now := time.Now()
	if newFS == nil {
		fsh.timeOfLastScan = now
		return
	}
	same := false
	if fsh.fileSystem != nil {
		same = CompareFileSystems(fsh.fileSystem, newFS, nil)
	}
	fsh.rwMutex.Lock()
	defer fsh.rwMutex.Unlock()
	fsh.durationOfLastScan = now.Sub(fsh.timeOfLastScan)
	scanTimeDistribution.Add(fsh.durationOfLastScan)
	fsh.scanCount++
	fsh.timeOfLastScan = now
	if fsh.fileSystem == nil {
		fsh.fileSystem = newFS
		fsh.generationCount = 1
		fsh.timeOfLastChange = fsh.timeOfLastScan
	} else {
		if !same {
			fsh.generationCount++
			fsh.fileSystem = newFS
			fsh.timeOfLastChange = fsh.timeOfLastScan
		}
	}
}

func (fsh *FileSystemHistory) updateObjectCacheOnly() error {
	if fsh.fileSystem == nil {
		return nil
	}
	oldObjectCache := fsh.fileSystem.ObjectCache
	if err := fsh.fileSystem.ScanObjectCache(); err != nil {
		return err
	}
	same := objectcache.CompareObjects(oldObjectCache,
		fsh.fileSystem.ObjectCache, nil)
	fsh.rwMutex.Lock()
	defer fsh.rwMutex.Unlock()
	fsh.scanCount++
	if !same {
		fsh.generationCount++
	}
	return nil
}
