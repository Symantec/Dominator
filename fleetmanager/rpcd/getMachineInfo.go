package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) GetMachineInfo(conn *srpc.Conn,
	request fm_proto.GetMachineInfoRequest,
	reply *fm_proto.GetMachineInfoResponse) error {
	if machine, subnets, err := t.getMachineInfo(request.Hostname); err != nil {
		*reply = fm_proto.GetMachineInfoResponse{
			Error: errors.ErrorToString(err)}
	} else {
		*reply = fm_proto.GetMachineInfoResponse{
			Machine: *machine,
			Subnets: subnets}
	}
	return nil
}

func (t *srpcType) getMachineInfo(hostname string) (
	*fm_proto.Machine, []*hyper_proto.Subnet, error) {
	topology, err := t.hypervisorsManager.GetTopology()
	if err != nil {
		return nil, nil, err
	}
	machine, err := t.hypervisorsManager.GetMachineInfo(hostname)
	if err != nil {
		return nil, nil, err
	}
	tSubnets, err := topology.GetSubnetsForMachine(hostname)
	if err != nil {
		return nil, nil, err
	}
	subnets := make([]*hyper_proto.Subnet, 0, len(tSubnets))
	for _, tSubnet := range tSubnets {
		subnets = append(subnets, &tSubnet.Subnet)
	}
	return &machine, subnets, err
}
