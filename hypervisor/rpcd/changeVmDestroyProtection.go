package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeVmDestroyProtection(conn *srpc.Conn,
	request hypervisor.ChangeVmDestroyProtectionRequest,
	reply *hypervisor.ChangeVmDestroyProtectionResponse) error {
	*reply = hypervisor.ChangeVmDestroyProtectionResponse{
		errors.ErrorToString(
			t.manager.ChangeVmDestroyProtection(request.IpAddress,
				conn.GetAuthInformation(),
				request.DestroyProtection))}
	return nil
}
