package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
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
