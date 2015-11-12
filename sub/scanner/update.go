package scanner

import (
	"github.com/Symantec/Dominator/lib/objectcache"
	"time"
)

func (fsh *FileSystemHistory) update(newFS *FileSystem) {
	now := time.Now()
	if newFS == nil {
		fsh.timeOfLastScan = now
		return
	}
	fsh.durationOfLastScan = now.Sub(fsh.timeOfLastScan)
	scanTimeDistribution.Add(fsh.durationOfLastScan.Seconds())
	fsh.scanCount++
	fsh.timeOfLastScan = now
	if fsh.fileSystem == nil {
		fsh.fileSystem = newFS
		fsh.generationCount = 1
		fsh.timeOfLastChange = fsh.timeOfLastScan
	} else {
		if !CompareFileSystems(fsh.fileSystem, newFS, nil) {
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
	if !objectcache.CompareObjects(oldObjectCache, fsh.fileSystem.ObjectCache,
		nil) {
		fsh.generationCount++
	}
	return nil
}
