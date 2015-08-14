package objectclient

import (
	"bytes"
	"errors"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
)

type myReadCloser struct {
	io.Reader
}

func (reader *myReadCloser) Close() error {
	return nil
}

func (objClient *ObjectClient) getObjects(hashes []hash.Hash) (
	*ObjectsReader, error) {
	var request objectserver.GetObjectsRequest
	request.Hashes = hashes
	var reply objectserver.GetObjectsResponse
	err := objClient.client.Call("ObjectServer.GetObjects", request, &reply)
	if err != nil {
		return nil, err
	}
	var objectsReader ObjectsReader
	objectsReader.nextIndex = -1
	objectsReader.sizes = reply.ObjectSizes
	objectsReader.datas = reply.Objects
	return &objectsReader, nil
}

func (or *ObjectsReader) nextObject() (uint64, io.ReadCloser, error) {
	or.nextIndex++
	if or.nextIndex >= int64(len(or.sizes)) {
		return 0, nil, errors.New("all objects have been consumed")
	}
	reader := &myReadCloser{bytes.NewReader(or.datas[or.nextIndex])}
	return or.sizes[or.nextIndex], reader, nil
}
