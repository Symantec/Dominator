package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) StartVm(conn *srpc.Conn,
	request hypervisor.StartVmRequest,
	reply *hypervisor.StartVmResponse) error {
	dhcpTimedOut, err := t.manager.StartVm(request.IpAddress,
		conn.GetAuthInformation(), request.AccessToken, request.DhcpTimeout)
	response := hypervisor.StartVmResponse{dhcpTimedOut,
		errors.ErrorToString(err)}
	*reply = response
	return nil
}
