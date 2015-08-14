package objectclient

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) addObjects(datas [][]byte,
	expectedHashes []*hash.Hash) ([]hash.Hash, error) {
	var request objectserver.AddObjectsRequest
	request.ObjectDatas = datas
	request.ExpectedHashes = expectedHashes
	var reply objectserver.AddObjectsResponse
	err := objClient.client.Call("ObjectServer.AddObjects", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Hashes, nil
}
