package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/fleetmanager/topology"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func getMachineInfoSubcommand(args []string, logger log.DebugLogger) error {
	err := getMachineInfo(args[0], logger)
	if err != nil {
		return fmt.Errorf("Error getting machine info: %s", err)
	}
	return nil
}

func getMachineInfo(hostname string, logger log.DebugLogger) error {
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	if info, err := getInfoForMachine(fmCR, hostname); err != nil {
		return err
	} else {
		return json.WriteWithIndent(os.Stdout, "    ", info)
	}
}

func getInfoForMachine(fmCR *srpc.ClientResource, hostname string) (
	fm_proto.GetMachineInfoResponse, error) {
	if *fleetManagerHostname != "" {
		return getInfoForMachineFromFleetManager(fmCR, hostname)
	}
	if info, err := getInfoForMachineFromTopology(hostname); err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	} else {
		return *info, nil
	}
}

func getInfoForMachineFromFleetManager(fmCR *srpc.ClientResource,
	hostname string) (fm_proto.GetMachineInfoResponse, error) {
	request := fm_proto.GetMachineInfoRequest{Hostname: hostname}
	var reply fm_proto.GetMachineInfoResponse
	client, err := fmCR.GetHTTP(nil, 0)
	if err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	defer client.Put()
	err = client.RequestReply("FleetManager.GetMachineInfo", request, &reply)
	if err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	if err := errors.New(reply.Error); err != nil {
		return fm_proto.GetMachineInfoResponse{}, err
	}
	return reply, nil
}

func getInfoForMachineFromTopology(hostname string) (
	*fm_proto.GetMachineInfoResponse, error) {
	if *topologyDir == "" {
		return nil, errors.New("no topologyDir specified")
	}
	topo, err := topology.Load(*topologyDir)
	if err != nil {
		return nil, err
	}
	machines, err := topo.ListMachines("")
	if err != nil {
		return nil, err
	}
	var machinePtr *fm_proto.Machine
	for _, machine := range machines {
		if machine.Hostname == hostname {
			machinePtr = machine
			break
		}
	}
	if machinePtr == nil {
		return nil,
			fmt.Errorf("machine: %s not found in topology", hostname)
	}
	subnets, err := topo.GetSubnetsForMachine(hostname)
	if err != nil {
		return nil, err
	}
	info := fm_proto.GetMachineInfoResponse{Machine: *machinePtr}
	info.Subnets = make([]*hyper_proto.Subnet, 0, len(subnets))
	for _, subnet := range subnets {
		info.Subnets = append(info.Subnets, &subnet.Subnet)
	}
	return &info, nil
}
