package objectclient

import (
	"bufio"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
	"io"
	"io/ioutil"
	"net"
)

func dial(network, address string) (net.Conn, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return conn, srpc.Call(conn, "ObjectServer.GetObjects")
}

func (objClient *ObjectClient) getObjects(hashes []hash.Hash) (
	*ObjectsReader, error) {
	conn, err := dial("tcp", objClient.address)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing\t%s\n", err))
	}
	var request objectserver.GetObjectsRequest
	var reply objectserver.GetObjectsResponse
	request.Exclusive = objClient.exclusiveGet
	request.Hashes = hashes
	encoder := gob.NewEncoder(conn)
	encoder.Encode(request)
	var objectsReader ObjectsReader
	objectsReader.conn = conn
	// Create a buffered reader and use it from now on, since otherwise
	// gob.NewDecoder() will create one under the covers and we could lose some
	// buffered data.
	objectsReader.reader = bufio.NewReader(conn)
	decoder := gob.NewDecoder(objectsReader.reader)
	err = decoder.Decode(&reply)
	if err != nil {
		return nil, err
	}
	if reply.ResponseString != "" {
		return nil, errors.New(reply.ResponseString)
	}
	objectsReader.nextIndex = -1
	objectsReader.sizes = reply.ObjectSizes
	return &objectsReader, nil
}

func (or *ObjectsReader) nextObject() (uint64, io.ReadCloser, error) {
	or.nextIndex++
	if or.nextIndex >= int64(len(or.sizes)) {
		return 0, nil, errors.New("all objects have been consumed")
	}
	size := or.sizes[or.nextIndex]
	return size,
		ioutil.NopCloser(&io.LimitedReader{or.reader, int64(size)}), nil
}
