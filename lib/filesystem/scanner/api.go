package scanner

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/fsrateio"
)

type FileSystem struct {
	rootDirectoryName       string
	fsScanContext           *fsrateio.ReaderContext
	scanFilter              *filter.Filter
	checkScanDisableRequest func() bool
	dev                     uint64
	inodeNumber             uint64
	filesystem.FileSystem
}

func ScanFileSystem(rootDirectoryName string,
	fsScanContext *fsrateio.ReaderContext, scanFilter *filter.Filter,
	checkScanDisableRequest func() bool, oldFS *FileSystem) (
	*FileSystem, error) {
	return scanFileSystem(rootDirectoryName, fsScanContext, scanFilter,
		checkScanDisableRequest, oldFS)
}
