package memory

import (
	"bytes"
	"errors"
	"github.com/Symantec/Dominator/lib/hash"
	"io"
	"io/ioutil"
)

func (objSrv *ObjectServer) getObjects(hashes []hash.Hash) (
	*ObjectsReader, error) {
	var objectsReader ObjectsReader
	objectsReader.objectServer = objSrv
	objectsReader.hashes = hashes
	objectsReader.nextIndex = -1
	return &objectsReader, nil
}

func (objSrv *ObjectServer) getObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	if data, ok := objSrv.objectMap[hashVal]; !ok {
		hashStr, _ := hashVal.MarshalText()
		return 0, nil, errors.New("missing object: " + string(hashStr))
	} else {
		return uint64(len(data)), ioutil.NopCloser(bytes.NewReader(data)), nil
	}
}

func (or *ObjectsReader) nextObject() (uint64, io.ReadCloser, error) {
	or.nextIndex++
	if or.nextIndex >= int64(len(or.hashes)) {
		return 0, nil, errors.New("all objects have been consumed")
	}
	return or.objectServer.getObject(or.hashes[or.nextIndex])
}
