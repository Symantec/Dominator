package util

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
)

type ComputedFile struct {
	Filename string
	Source   string
}

type ComputedFilesData struct {
	FileData      map[string][]byte // Key: filename.
	RootDirectory string
}

// CopyMtimes will copy modification times for files from the source to the
// destination if the file data and metadata (other than mtime) are identical.
// Directory entry inode pointers are invalidated by this operation, so this
// should be followed by a call to dest.RebuildInodePointers().
func CopyMtimes(source, dest *filesystem.FileSystem) {
	copyMtimes(source, dest)
}

func LoadComputedFiles(filename string) ([]ComputedFile, error) {
	return loadComputedFiles(filename)
}

func ReplaceComputedFiles(fs *filesystem.FileSystem,
	computedFilesData *ComputedFilesData,
	objectsGetter objectserver.ObjectsGetter) (
	objectserver.ObjectsGetter, error) {
	return replaceComputedFiles(fs, computedFilesData, objectsGetter)
}

func SpliceComputedFiles(fs *filesystem.FileSystem,
	computedFileList []ComputedFile) error {
	return spliceComputedFiles(fs, computedFileList)
}

func Unpack(fs *filesystem.FileSystem, objectsGetter objectserver.ObjectsGetter,
	rootDir string, logger log.Logger) error {
	return unpack(fs, objectsGetter, rootDir, logger)
}
