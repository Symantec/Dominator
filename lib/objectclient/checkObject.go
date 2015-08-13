package objectclient

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) checkObject(hashVal hash.Hash) (bool, error) {
	var request objectserver.CheckObjectsRequest
	request.Objects = make([]hash.Hash, 1)
	request.Objects[0] = hashVal
	var reply objectserver.CheckObjectsResponse
	err := objClient.client.Call("ObjectServer.CheckObjects", request, &reply)
	if err != nil {
		return false, err
	}
	return reply.ObjectsPresent[0], nil
}
