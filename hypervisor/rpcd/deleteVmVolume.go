package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) DeleteVmVolume(conn *srpc.Conn,
	request hypervisor.DeleteVmVolumeRequest,
	reply *hypervisor.DeleteVmVolumeResponse) error {
	*reply = hypervisor.DeleteVmVolumeResponse{
		errors.ErrorToString(t.manager.DeleteVmVolume(request.IpAddress,
			conn.GetAuthInformation(), request.AccessToken,
			request.VolumeIndex))}
	return nil
}
