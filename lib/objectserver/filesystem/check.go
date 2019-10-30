package filesystem

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
)

func (objSrv *ObjectServer) checkObjects(hashes []hash.Hash) ([]uint64, error) {
	sizesList := make([]uint64, len(hashes))
	for index, hash := range hashes {
		var err error
		sizesList[index], err = objSrv.checkObject(hash)
		if err != nil {
			return nil, err
		}
	}
	return sizesList, nil
}

func (objSrv *ObjectServer) checkObject(hash hash.Hash) (uint64, error) {
	objSrv.rwLock.RLock()
	size, ok := objSrv.sizesMap[hash]
	objSrv.rwLock.RUnlock()
	if ok {
		return size, nil
	}
	filename := path.Join(objSrv.baseDir, objectcache.HashToFilename(hash))
	fi, err := os.Lstat(filename)
	if err != nil {
		return 0, nil
	}
	if fi.Mode().IsRegular() {
		if fi.Size() < 1 {
			return 0, errors.New(fmt.Sprintf("zero length file: %s", filename))
		}
		size := uint64(fi.Size())
		objSrv.rwLock.Lock()
		objSrv.sizesMap[hash] = size
		objSrv.rwLock.Unlock()
		return size, nil
	}
	return 0, nil
}
