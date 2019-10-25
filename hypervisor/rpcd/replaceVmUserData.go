package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ReplaceVmUserData(conn *srpc.Conn,
	request hypervisor.ReplaceVmUserDataRequest,
	reply *hypervisor.ReplaceVmUserDataResponse) error {
	response := hypervisor.ReplaceVmUserDataResponse{
		errors.ErrorToString(t.manager.ReplaceVmUserData(request.IpAddress,
			conn, request.Size, conn.GetAuthInformation()))}
	*reply = response
	return nil
}
