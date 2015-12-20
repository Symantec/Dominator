package objectclient

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/objectserver"
	"io"
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
	client    *srpc.Client
	reader    io.Reader
	nextIndex int64
}

func (or *ObjectsReader) Close() error {
	return or.client.Close()
}

func (or *ObjectsReader) NextObject() (uint64, io.ReadCloser, error) {
	return or.nextObject()
}

type ObjectAdderQueue struct {
	client          *srpc.Client
	conn            *srpc.Conn
	encoder         *gob.Encoder
	getResponseChan chan<- bool
	errorChan       <-chan error
}

func NewObjectAdderQueue(objClient *ObjectClient) (*ObjectAdderQueue, error) {
	return newObjectAdderQueue(objClient)
}

func (objQ *ObjectAdderQueue) Add(reader io.Reader, length uint64) (
	hash.Hash, error) {
	return objQ.add(reader, length)
}

func (objQ *ObjectAdderQueue) Close() error {
	return objQ.close()
}
