package memory

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
)

func (objSrv *ObjectServer) getData(hashVal hash.Hash) ([]byte, error) {
	objSrv.rwLock.RLock()
	defer objSrv.rwLock.RUnlock()
	if data, ok := objSrv.objectMap[hashVal]; !ok {
		hashStr, _ := hashVal.MarshalText()
		return nil, errors.New("missing object: " + string(hashStr))
	} else {
		return data, nil
	}
}

func (objSrv *ObjectServer) getObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	if data, err := objSrv.getData(hashVal); err != nil {
		return 0, nil, err
	} else {
		return uint64(len(data)), ioutil.NopCloser(bytes.NewReader(data)), nil
	}
}

func (objSrv *ObjectServer) getObjects(hashes []hash.Hash) (
	*ObjectsReader, error) {
	objectsReader := ObjectsReader{
		objectServer: objSrv,
		hashes:       hashes,
		nextIndex:    -1,
		sizes:        make([]uint64, 0, len(hashes)),
	}
	for _, hashVal := range hashes {
		data, err := objSrv.getData(hashVal)
		if err != nil {
			return nil, err
		}
		objectsReader.sizes = append(objectsReader.sizes, uint64(len(data)))
	}
	return &objectsReader, nil
}

func (or *ObjectsReader) nextObject() (uint64, io.ReadCloser, error) {
	or.nextIndex++
	if or.nextIndex >= int64(len(or.hashes)) {
		return 0, nil, errors.New("all objects have been consumed")
	}
	return or.objectServer.getObject(or.hashes[or.nextIndex])
}
