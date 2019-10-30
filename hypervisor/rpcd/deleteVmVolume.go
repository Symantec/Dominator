package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
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
