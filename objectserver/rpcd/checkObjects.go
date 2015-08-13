package rpcd

import (
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (t *rpcType) CheckObjects(request objectserver.CheckObjectsRequest,
	reply *objectserver.CheckObjectsResponse) error {
	var response objectserver.CheckObjectsResponse
	response.ObjectsPresent = make([]bool, len(request.Objects))
	for index, hash := range request.Objects {
		response.ObjectsPresent[index] = objectServer.CheckObject(hash)
	}
	*reply = response
	return nil
}
