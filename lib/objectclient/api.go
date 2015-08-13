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

func (objClient *ObjectClient) AddObject(data []byte, expectedHash *hash.Hash) (
	hash.Hash, error) {
	return objClient.addObject(data, expectedHash)
}

func (objClient *ObjectClient) CheckObject(hash hash.Hash) (bool, error) {
	return objClient.checkObject(hash)
}

func (objClient *ObjectClient) GetObjectReader(hash hash.Hash) (uint64,
	io.ReadCloser, error) {
	return objClient.getObjectReader(hash)
}
