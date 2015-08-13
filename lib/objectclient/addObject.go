package objectclient

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) addObject(data []byte, expectedHash *hash.Hash) (
	hash.Hash, error) {
	var request objectserver.AddObjectsRequest
	request.ObjectsToAdd = make([]*objectserver.AddObjectSubrequest, 1)
	request.ObjectsToAdd[0].ObjectData = data
	request.ObjectsToAdd[0].ExpectedHash = expectedHash
	var reply objectserver.AddObjectsResponse
	err := objClient.client.Call("ObjectServer.AddObjects", request, &reply)
	if err != nil {
		return reply.Hashes[0], err
	}
	return reply.Hashes[0], nil
}
