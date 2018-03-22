package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) RestoreVmImage(conn *srpc.Conn,
	request hypervisor.RestoreVmImageRequest,
	reply *hypervisor.RestoreVmImageResponse) error {
	response := hypervisor.RestoreVmImageResponse{
		errors.ErrorToString(t.manager.RestoreVmImage(request.IpAddress,
			conn.GetAuthInformation()))}
	*reply = response
	return nil
}
