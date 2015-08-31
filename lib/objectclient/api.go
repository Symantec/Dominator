package objectclient

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/objectserver"
	"io"
	"net"
)

type ObjectClient struct {
	address      string
	exclusiveGet bool
}

func NewObjectClient(address string) *ObjectClient {
	return &ObjectClient{address, false}
}

func (objClient *ObjectClient) AddObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	return objClient.addObjects(datas, expectedHashes)
}

func (objClient *ObjectClient) CheckObjects(hashes []hash.Hash) (
	[]uint64, error) {
	return objClient.checkObjects(hashes)
}

func (objClient *ObjectClient) GetObjects(hashes []hash.Hash) (
	objectserver.ObjectsReader, error) {
	return objClient.getObjects(hashes)
}

func (objClient *ObjectClient) SetExclusiveGetObjects(exclusive bool) {
	objClient.exclusiveGet = exclusive
}

type ObjectsReader struct {
	sizes     []uint64
	conn      net.Conn
	nextIndex int64
}

func (or *ObjectsReader) NextObject() (uint64, io.ReadCloser, error) {
	return or.nextObject()
}

type ObjectAdderQueue struct {
	numBytes       uint64
	maxBytes       uint64
	client         *ObjectClient
	datas          [][]byte
	expectedHashes []*hash.Hash
}

func NewObjectAdderQueue(client *ObjectClient,
	maxBytes uint64) *ObjectAdderQueue {
	return &ObjectAdderQueue{client: client, maxBytes: maxBytes}
}

func (objQ *ObjectAdderQueue) Add(data []byte) (hash.Hash, error) {
	return objQ.add(data)
}

func (objQ *ObjectAdderQueue) Flush() error {
	return objQ.flush()
}
