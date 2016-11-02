package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

func (objSrv *ObjectServer) deleteObject(hashVal hash.Hash) error {
	filename := path.Join(objSrv.baseDir, objectcache.HashToFilename(hashVal))
	if err := os.Remove(filename); err != nil {
		return err
	}
	objSrv.rwLock.Lock()
	delete(objSrv.sizesMap, hashVal)
	objSrv.rwLock.Unlock()
	return nil
}
