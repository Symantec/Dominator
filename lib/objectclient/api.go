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

func (objSrv *ObjectClient) AddObject(data []byte, expectedHash *hash.Hash) (
	hash.Hash, error) {
	return objSrv.addObject(data, expectedHash)
}

func (objSrv *ObjectClient) CheckObject(hash hash.Hash) (bool, error) {
	return objSrv.checkObject(hash)
}

func (objSrv *ObjectClient) GetObjectReader(hash hash.Hash) (uint64,
	io.ReadCloser, error) {
	return objSrv.getObjectReader(hash)
}
