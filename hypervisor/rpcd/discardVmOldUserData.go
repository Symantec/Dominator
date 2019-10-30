package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) DiscardVmOldUserData(conn *srpc.Conn,
	request hypervisor.DiscardVmOldUserDataRequest,
	reply *hypervisor.DiscardVmOldUserDataResponse) error {
	response := hypervisor.DiscardVmOldUserDataResponse{
		errors.ErrorToString(t.manager.DiscardVmOldUserData(request.IpAddress,
			conn.GetAuthInformation()))}
	*reply = response
	return nil
}
