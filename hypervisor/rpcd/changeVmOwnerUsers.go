package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeVmOwnerUsers(conn *srpc.Conn,
	request hypervisor.ChangeVmOwnerUsersRequest,
	reply *hypervisor.ChangeVmOwnerUsersResponse) error {
	response := hypervisor.ChangeVmOwnerUsersResponse{
		errors.ErrorToString(
			t.manager.ChangeVmOwnerUsers(request.IpAddress,
				conn.GetAuthInformation(),
				request.OwnerUsers))}
	*reply = response
	return nil
}
