package scanner

import (
	"regexp"
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

func (configuration *Configuration) setExclusionList(reList []string) error {
	exclusionList := make([]*regexp.Regexp, len(reList))
	for index, reEntry := range reList {
		var err error
		exclusionList[index], err = regexp.Compile("^" + reEntry)
		if err != nil {
			return err
		}
	}
	configuration.ExclusionList = exclusionList
	return nil
}
