package cachingreader

import (
	"os"
	"path/filepath"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
)

func (objSrv *ObjectServer) linkObject(filename string,
	hashVal hash.Hash) (bool, error) {
	objectsReader, err := objSrv.GetObjects([]hash.Hash{hashVal})
	if err != nil {
		return false, err
	}
	defer objectsReader.Close()
	size, reader, err := objectsReader.NextObject()
	if err != nil {
		return false, err
	}
	defer reader.Close()
	source := filepath.Join(objSrv.baseDir, objectcache.HashToFilename(hashVal))
	if err := os.Link(source, filename); err == nil {
		return true, nil
	}
	return false, fsutil.CopyToFile(filename, privateFilePerms, reader, size)
}
