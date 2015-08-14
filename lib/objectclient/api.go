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
