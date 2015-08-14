package objectclient

import (
	"bytes"
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

func (objClient *ObjectClient) getObjectReader(hashVal hash.Hash) (uint64,
	io.ReadCloser, error) {
	var request objectserver.GetObjectsRequest
	request.Hashes = make([]hash.Hash, 1)
	request.Hashes[0] = hashVal
	var reply objectserver.GetObjectsResponse
	err := objClient.client.Call("ObjectServer.GetObjects", request, &reply)
	if err != nil {
		return 0, nil, err
	}
	reader := &myReadCloser{bytes.NewReader(reply.Objects[0])}
	return reply.ObjectSizes[0], reader, nil
}
