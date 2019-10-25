package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func (t *srpcType) ListHypervisorsInLocation(conn *srpc.Conn,
	request proto.ListHypervisorsInLocationRequest,
	reply *proto.ListHypervisorsInLocationResponse) error {
	addresses, err := t.hypervisorsManager.ListHypervisorsInLocation(request)
	*reply = proto.ListHypervisorsInLocationResponse{
		HypervisorAddresses: addresses,
		Error:               errors.ErrorToString(err),
	}
	return nil
}
