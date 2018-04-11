package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
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
