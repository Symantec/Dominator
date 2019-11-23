package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ListSubnets(conn *srpc.Conn,
	request hypervisor.ListSubnetsRequest,
	reply *hypervisor.ListSubnetsResponse) error {
	*reply = hypervisor.ListSubnetsResponse{
		Subnets: t.manager.ListSubnets(request.Sort)}
	return nil
}
