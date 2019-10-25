package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) DiscardVmOldImage(conn *srpc.Conn,
	request hypervisor.DiscardVmOldImageRequest,
	reply *hypervisor.DiscardVmOldImageResponse) error {
	response := hypervisor.DiscardVmOldImageResponse{
		errors.ErrorToString(t.manager.DiscardVmOldImage(request.IpAddress,
			conn.GetAuthInformation()))}
	*reply = response
	return nil
}
