package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

func (objSrv *FileSystemObjectServer) checkObject(hash hash.Hash) bool {
	if objSrv.checkMap[hash] {
		return true
	}
	filename := path.Join(objSrv.topDirectory, objectcache.HashToFilename(hash))
	fi, err := os.Lstat(filename)
	if err != nil {
		return false
	}
	if fi.Mode().IsRegular() {
		objSrv.checkMap[hash] = true
		return true
	}
	return false
}
