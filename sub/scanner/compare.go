package scanner

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
)

func compareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	if !filesystem.CompareFileSystems(&left.FileSystem, &right.FileSystem,
		logWriter) {
		return false
	}
	return objectcache.CompareObjects(left.ObjectCache, right.ObjectCache,
		logWriter)
}
