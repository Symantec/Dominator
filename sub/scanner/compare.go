package scanner

import (
	"io"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectcache"
)

func compareFileSystems(left, right *FileSystem, logWriter io.Writer) bool {
	if !filesystem.CompareFileSystems(&left.FileSystem.FileSystem,
		&right.FileSystem.FileSystem, logWriter) {
		return false
	}
	return objectcache.CompareObjects(left.ObjectCache, right.ObjectCache,
		logWriter)
}
