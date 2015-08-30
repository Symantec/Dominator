package objectclient

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
	"net/rpc"
)

func (objClient *ObjectClient) addObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	for _, data := range datas {
		if len(data) < 1 {
			return nil, errors.New("zero length object cannot be added")
		}
	}
	var request objectserver.AddObjectsRequest
	request.ObjectDatas = datas
	request.ExpectedHashes = expectedHashes
	var reply objectserver.AddObjectsResponse
	client, err := rpc.DialHTTP("tcp", objClient.address)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing\t%s\n", err.Error()))
	}
	defer client.Close()
	err = client.Call("ObjectServer.AddObjects", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Hashes, nil
}
