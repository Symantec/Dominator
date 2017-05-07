package filesystem

import (
	"errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"os"
	"path"
	"syscall"
)

var stashDirectory string = ".stash"

func (objSrv *ObjectServer) commitObject(hashVal hash.Hash) error {
	hashName := objectcache.HashToFilename(hashVal)
	filename := path.Join(objSrv.baseDir, hashName)
	stashFilename := path.Join(objSrv.baseDir, stashDirectory, hashName)
	fi, err := os.Lstat(stashFilename)
	if err != nil {
		return err
	}
	if !fi.Mode().IsRegular() {
		fsutil.ForceRemove(stashFilename)
		return errors.New("Existing non-file: " + stashFilename)
	}
	if err = os.MkdirAll(path.Dir(filename), syscall.S_IRWXU); err != nil {
		return err
	}
	objSrv.rwLock.Lock()
	defer objSrv.rwLock.Unlock()
	if _, ok := objSrv.sizesMap[hashVal]; ok {
		fsutil.ForceRemove(stashFilename)
		return nil
	} else {
		objSrv.sizesMap[hashVal] = uint64(fi.Size())
		return os.Rename(stashFilename, filename)
	}
}

func (objSrv *ObjectServer) deleteStashedObject(hashVal hash.Hash) error {
	filename := path.Join(objSrv.baseDir, stashDirectory,
		objectcache.HashToFilename(hashVal))
	return os.Remove(filename)
}

func (objSrv *ObjectServer) stashOrVerifyObject(reader io.Reader,
	length uint64, expectedHash *hash.Hash) (hash.Hash, []byte, error) {
	hashVal, data, err := objectcache.ReadObject(reader, length, expectedHash)
	if err != nil {
		return hashVal, nil, err
	}
	length = uint64(len(data))
	hashName := objectcache.HashToFilename(hashVal)
	filename := path.Join(objSrv.baseDir, hashName)
	// Check for existing object and collision.
	if length, err := objSrv.checkObject(hashVal); err != nil {
		return hashVal, nil, err
	} else if length > 0 {
		if err := collisionCheck(data, filename, int64(length)); err != nil {
			return hashVal, nil, err
		}
		return hashVal, nil, nil
	}
	// Check for existing stashed object and collision.
	stashFilename := path.Join(objSrv.baseDir, stashDirectory, hashName)
	if _, err := addOrCompare(hashVal, data, stashFilename); err != nil {
		return hashVal, nil, err
	} else {
		return hashVal, data, nil
	}
}
