package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeVmSize(conn *srpc.Conn,
	request hypervisor.ChangeVmSizeRequest,
	reply *hypervisor.ChangeVmSizeResponse) error {
	*reply = hypervisor.ChangeVmSizeResponse{
		errors.ErrorToString(
			t.manager.ChangeVmSize(request.IpAddress, conn.GetAuthInformation(),
				request.MemoryInMiB, request.MilliCPUs))}
	return nil
}
