package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) UpdateSubnets(conn *srpc.Conn,
	request hypervisor.UpdateSubnetsRequest,
	reply *hypervisor.UpdateSubnetsResponse) error {
	*reply = hypervisor.UpdateSubnetsResponse{
		errors.ErrorToString(t.manager.UpdateSubnets(request))}
	return nil
}
