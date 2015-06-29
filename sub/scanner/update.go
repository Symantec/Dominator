package scanner

import (
	"time"
)

func (fsh *FileSystemHistory) update(newFS *FileSystem) {
	now := time.Now()
	if newFS == nil {
		fsh.timeOfLastScan = now
		return
	}
	fsh.durationOfLastScan = now.Sub(fsh.timeOfLastScan)
	fsh.scanCount++
	fsh.timeOfLastScan = now
	if fsh.fileSystem == nil {
		fsh.fileSystem = newFS
		fsh.generationCount = 1
		fsh.timeOfLastChange = fsh.timeOfLastScan
	} else {
		if !Compare(fsh.fileSystem, newFS, nil) {
			fsh.generationCount++
			fsh.fileSystem = newFS
			fsh.timeOfLastChange = fsh.timeOfLastScan
		}
	}
}
