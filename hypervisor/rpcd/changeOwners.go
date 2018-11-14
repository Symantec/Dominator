package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeOwners(conn *srpc.Conn,
	request hypervisor.ChangeOwnersRequest,
	reply *hypervisor.ChangeOwnersResponse) error {
	response := hypervisor.ChangeOwnersResponse{
		errors.ErrorToString(
			t.manager.ChangeOwners(request.OwnerGroups, request.OwnerUsers))}
	*reply = response
	return nil
}
