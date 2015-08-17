package objectclient

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) checkObjects(hashes []hash.Hash) (
	[]uint64, error) {
	var request objectserver.CheckObjectsRequest
	request.Hashes = hashes
	var reply objectserver.CheckObjectsResponse
	err := objClient.client.Call("ObjectServer.CheckObjects", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.ObjectSizes, nil
}
