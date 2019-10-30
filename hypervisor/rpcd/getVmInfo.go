package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) GetVmInfo(conn *srpc.Conn,
	request hypervisor.GetVmInfoRequest,
	reply *hypervisor.GetVmInfoResponse) error {
	info, err := t.manager.GetVmInfo(request.IpAddress)
	response := hypervisor.GetVmInfoResponse{
		VmInfo: info,
		Error:  errors.ErrorToString(err),
	}
	*reply = response
	return nil
}
