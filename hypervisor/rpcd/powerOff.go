package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) PowerOff(conn *srpc.Conn,
	request hypervisor.PowerOffRequest,
	reply *hypervisor.PowerOffResponse) error {
	*reply = hypervisor.PowerOffResponse{
		Error: errors.ErrorToString(t.manager.PowerOff(request.StopVMs))}
	return nil
}
