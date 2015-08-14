package objectclient

import (
	"github.com/Symantec/Dominator/lib/hash"
	"io"
	"net/rpc"
)

type ObjectClient struct {
	client *rpc.Client
}

func NewObjectClient(client *rpc.Client) *ObjectClient {
	return &ObjectClient{client}
}

func (objClient *ObjectClient) AddObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	return objClient.addObjects(datas, expectedHashes)
}

func (objClient *ObjectClient) CheckObjects(hashes []hash.Hash) (
	[]bool, error) {
	return objClient.checkObjects(hashes)
}

func (objClient *ObjectClient) GetObjectReader(hash hash.Hash) (uint64,
	io.ReadCloser, error) {
	return objClient.getObjectReader(hash)
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

func (objQ *ObjectAdderQueue) Add(data []byte) error {
	return objQ.add(data)
}

func (objQ *ObjectAdderQueue) Flush() error {
	return objQ.flush()
}
