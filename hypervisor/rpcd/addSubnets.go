package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) AddSubnets(conn *srpc.Conn,
	request hypervisor.AddSubnetsRequest,
	reply *hypervisor.AddSubnetsResponse) error {
	response := hypervisor.AddSubnetsResponse{
		errors.ErrorToString(t.manager.AddSubnets(request.Subnets))}
	*reply = response
	return nil
}
