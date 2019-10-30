package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) UpdateSubnets(conn *srpc.Conn,
	request hypervisor.UpdateSubnetsRequest,
	reply *hypervisor.UpdateSubnetsResponse) error {
	*reply = hypervisor.UpdateSubnetsResponse{
		errors.ErrorToString(t.manager.UpdateSubnets(request))}
	return nil
}
