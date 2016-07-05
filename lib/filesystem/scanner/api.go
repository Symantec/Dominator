package scanner

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/fsrateio"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

type Hasher interface {
	Hash(reader io.Reader, length uint64) (hash.Hash, error)
}

type FileSystem struct {
	rootDirectoryName       string
	fsScanContext           *fsrateio.ReaderContext
	scanFilter              *filter.Filter
	checkScanDisableRequest func() bool
	hasher                  Hasher
	dev                     uint64
	inodeNumber             uint64
	filesystem.FileSystem
}

func ScanFileSystem(rootDirectoryName string,
	fsScanContext *fsrateio.ReaderContext, scanFilter *filter.Filter,
	checkScanDisableRequest func() bool, hasher Hasher, oldFS *FileSystem) (
	*FileSystem, error) {
	return scanFileSystem(rootDirectoryName, fsScanContext, scanFilter,
		checkScanDisableRequest, hasher, oldFS)
}

func (fs *FileSystem) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return fs.getObject(hashVal)
}
