package rpcd

import (
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objSrv *objectServer) CheckObjects(
	request objectserver.CheckObjectsRequest,
	reply *objectserver.CheckObjectsResponse) error {
	var response objectserver.CheckObjectsResponse
	var err error
	response.ObjectSizes, err = objSrv.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		return err
	}
	*reply = response
	return nil
}
