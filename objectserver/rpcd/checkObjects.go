package rpcd

import (
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (t *rpcType) CheckObjects(request objectserver.CheckObjectsRequest,
	reply *objectserver.CheckObjectsResponse) error {
	var response objectserver.CheckObjectsResponse
	response.ObjectsPresent = make([]bool, len(request.Objects))
	for index, hash := range request.Objects {
		var err error
		response.ObjectsPresent[index], err = objectServer.CheckObject(hash)
		if err != nil {
			return err
		}
	}
	*reply = response
	return nil
}
