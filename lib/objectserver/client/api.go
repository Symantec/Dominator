package client

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	"io"
)

type ObjectClient struct {
	address      string
	client       *srpc.Client
	exclusiveGet bool
}

func NewObjectClient(address string) *ObjectClient {
	return &ObjectClient{address: address}
}

func AttachObjectClient(client *srpc.Client) *ObjectClient {
	return &ObjectClient{client: client}
}

func (objClient *ObjectClient) AddObject(reader io.Reader, length uint64,
	expectedHash *hash.Hash) (hash.Hash, bool, error) {
	return objClient.addObject(reader, length, expectedHash)
}

func (objClient *ObjectClient) CheckObjects(hashes []hash.Hash) (
	[]uint64, error) {
	return objClient.checkObjects(hashes)
}

func (objClient *ObjectClient) Close() error {
	return objClient.close()
}

func (objClient *ObjectClient) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return objectserver.GetObject(objClient, hashVal)
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
	conn            *srpc.Conn
	encoder         *gob.Encoder
	getResponseChan chan<- bool
	errorChan       <-chan error
	sendSemaphore   chan bool
}

func NewObjectAdderQueue(client *srpc.Client) (*ObjectAdderQueue, error) {
	return newObjectAdderQueue(client)
}

func (objQ *ObjectAdderQueue) Add(reader io.Reader, length uint64) (
	hash.Hash, error) {
	return objQ.add(reader, length)
}

func (objQ *ObjectAdderQueue) Close() error {
	return objQ.close()
}
