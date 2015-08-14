package filesystem

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"os"
	"path"
)

func (objSrv *FileSystemObjectServer) checkObjects(hashes []hash.Hash) (
	[]bool, error) {
	presentList := make([]bool, len(hashes))
	for index, hash := range hashes {
		var err error
		presentList[index], err = objSrv.checkObject(hash)
		if err != nil {
			return nil, err
		}
	}
	return presentList, nil
}

func (objSrv *FileSystemObjectServer) checkObject(hash hash.Hash) (
	bool, error) {
	if objSrv.checkMap[hash] {
		return true, nil
	}
	filename := path.Join(objSrv.baseDir, objectcache.HashToFilename(hash))
	fi, err := os.Lstat(filename)
	if err != nil {
		return false, nil
	}
	if fi.Mode().IsRegular() {
		objSrv.checkMap[hash] = true
		return true, nil
	}
	return false, nil
}
