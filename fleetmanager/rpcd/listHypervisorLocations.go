package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func (t *srpcType) ListHypervisorLocations(conn *srpc.Conn,
	request proto.ListHypervisorLocationsRequest,
	reply *proto.ListHypervisorLocationsResponse) error {
	locations, err := t.hypervisorsManager.ListLocations(request.TopLocation)
	*reply = proto.ListHypervisorLocationsResponse{
		Locations: locations,
		Error:     errors.ErrorToString(err),
	}
	return nil
}
