package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
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
