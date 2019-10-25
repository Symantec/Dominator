package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func (t *srpcType) MoveIpAddresses(conn *srpc.Conn,
	request fm_proto.MoveIpAddressesRequest,
	reply *fm_proto.MoveIpAddressesResponse) error {
	err := t.hypervisorsManager.MoveIpAddresses(request.HypervisorHostname,
		request.IpAddresses)
	if err != nil {
		*reply = fm_proto.MoveIpAddressesResponse{
			Error: errors.ErrorToString(err)}
	}
	return nil
}
