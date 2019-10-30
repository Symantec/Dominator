package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func (t *srpcType) ChangeMachineTags(conn *srpc.Conn,
	request fleetmanager.ChangeMachineTagsRequest,
	reply *fleetmanager.ChangeMachineTagsResponse) error {
	*reply = fleetmanager.ChangeMachineTagsResponse{
		errors.ErrorToString(t.hypervisorsManager.ChangeMachineTags(
			request.Hostname, conn.GetAuthInformation(), request.Tags))}
	return nil
}
