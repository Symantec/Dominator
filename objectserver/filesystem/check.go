package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

func (objSrv *FileSystemObjectServer) checkObject(hash hash.Hash) bool {
	filename := path.Join(objSrv.topDirectory, objectcache.HashToFilename(hash))
	fi, err := os.Lstat(filename)
	if err != nil {
		return false
	}
	return fi.Mode().IsRegular()
}
