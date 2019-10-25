package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
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
