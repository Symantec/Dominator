package objectserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
)

func getObject(objSrv ObjectServer, hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	hashes := make([]hash.Hash, 1)
	hashes[0] = hashVal
	objectsReader, err := objSrv.GetObjects(hashes)
	if err != nil {
		return 0, nil, err
	}
	defer objectsReader.Close()
	size, reader, err := objectsReader.NextObject()
	if err != nil {
		return 0, nil, err
	}
	return size, reader, nil
}
