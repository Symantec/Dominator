package util

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver"
)

// CopyMtimes will copy modification times for files from the source to the
// destination if the file data and metadata (other than mtime) are identical.
// Directory entry inode pointers are invalidated by this operation, so this
// should be followed by a call to dest.RebuildInodePointers().
func CopyMtimes(source, dest *filesystem.FileSystem) {
	copyMtimes(source, dest)
}

func Unpack(fs *filesystem.FileSystem, objectsGetter objectserver.ObjectsGetter,
	rootDir string, logger log.Logger) error {
	return unpack(fs, objectsGetter, rootDir, logger)
}
