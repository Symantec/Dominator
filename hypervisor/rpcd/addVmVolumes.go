package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) AddVmVolumes(conn *srpc.Conn,
	request hypervisor.AddVmVolumesRequest,
	reply *hypervisor.AddVmVolumesResponse) error {
	*reply = hypervisor.AddVmVolumesResponse{
		errors.ErrorToString(t.manager.AddVmVolumes(request.IpAddress,
			conn.GetAuthInformation(), request.VolumeSizes))}
	return nil
}
