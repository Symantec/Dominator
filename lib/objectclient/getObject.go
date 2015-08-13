package objectclient

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
)

type myReadCloser struct {
	reader io.Reader
}

func (reader *myReadCloser) Read(b []byte) (int, error) {
	return reader.Read(b)
}

func (reader *myReadCloser) Close() error {
	return nil
}

func (objSrv *ObjectClient) getObjectReader(hashVal hash.Hash) (uint64,
	io.ReadCloser, error) {
	var request objectserver.GetObjectsRequest
	request.Objects = make([]hash.Hash, 1)
	request.Objects[0] = hashVal
	var reply objectserver.GetObjectsResponse
	err := objSrv.client.Call("ObjectServer.GetObjects", request, &reply)
	if err != nil {
		return 0, nil, err
	}
	reader := &myReadCloser{bytes.NewReader(reply.Objects[0])}
	return reply.ObjectSizes[0], reader, nil
}
