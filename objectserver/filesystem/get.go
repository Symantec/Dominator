package filesystem

import (
	"errors"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"os"
	"path"
)

func (objSrv *ObjectServer) getObjects(hashes []hash.Hash) (
	*ObjectsReader, error) {
	var objectsReader ObjectsReader
	objectsReader.objectServer = objSrv
	objectsReader.hashes = hashes
	objectsReader.nextIndex = -1
	return &objectsReader, nil
}

func (or *ObjectsReader) nextObject() (uint64, io.ReadCloser, error) {
	or.nextIndex++
	if or.nextIndex >= int64(len(or.hashes)) {
		return 0, nil, errors.New("all objects have been consumed")
	}
	filename := path.Join(or.objectServer.baseDir,
		objectcache.HashToFilename(or.hashes[or.nextIndex]))
	file, err := os.Open(filename)
	if err != nil {
		return 0, nil, err
	}
	fi, err := file.Stat()
	if err != nil {
		file.Close()
		return 0, nil, err
	}
	return uint64(fi.Size()), file, nil
}
