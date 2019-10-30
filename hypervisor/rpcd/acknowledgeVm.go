package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) AcknowledgeVm(conn *srpc.Conn,
	request hypervisor.AcknowledgeVmRequest,
	reply *hypervisor.AcknowledgeVmResponse) error {
	response := hypervisor.AcknowledgeVmResponse{
		errors.ErrorToString(t.manager.AcknowledgeVm(request.IpAddress,
			conn.GetAuthInformation()))}
	*reply = response
	return nil
}
