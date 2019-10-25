package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeVmConsoleType(conn *srpc.Conn,
	request hypervisor.ChangeVmConsoleTypeRequest,
	reply *hypervisor.ChangeVmConsoleTypeResponse) error {
	*reply = hypervisor.ChangeVmConsoleTypeResponse{
		errors.ErrorToString(
			t.manager.ChangeVmConsoleType(request.IpAddress,
				conn.GetAuthInformation(),
				request.ConsoleType))}
	return nil
}
