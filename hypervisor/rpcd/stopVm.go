package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) StopVm(conn *srpc.Conn,
	request hypervisor.StopVmRequest, reply *hypervisor.StopVmResponse) error {
	response := hypervisor.StopVmResponse{
		errors.ErrorToString(t.manager.StopVm(request.IpAddress,
			conn.GetAuthInformation(), request.AccessToken))}
	*reply = response
	return nil
}
