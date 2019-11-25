package main

import (
	"fmt"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	fm_proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func netbootVmSubcommand(args []string, logger log.DebugLogger) error {
	err := netbootVm(logger)
	if err != nil {
		return fmt.Errorf("Error netbooting VM: %s", err)
	}
	return nil
}

func netbootVm(logger log.DebugLogger) error {
	if len(subnetIDs) < 1 {
		return errors.New("no subnetIDs specified")
	}
	fmCR := srpc.NewClientResource("tcp",
		fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum))
	defer fmCR.ScheduleClose()
	imageClient, err := srpc.DialHTTP("tcp", fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum), 0)
	if err != nil {
		return fmt.Errorf("%s: %s", *imageServerHostname, err)
	}
	defer imageClient.Close()
	var hypervisorAddresses []string
	if *hypervisorHostname != "" {
		hypervisorAddresses = append(hypervisorAddresses,
			fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum))
	} else {
		hypervisorAddresses, err = listConnectedHypervisorsInLocation(fmCR,
			*location)
		if err != nil {
			return err
		}
	}
	if len(hypervisorAddresses) < 1 {
		return errors.New("no nearby Hypervisors available")
	}
	logger.Debugf(0, "Selected %s as boot server on subnet: %s\n",
		hypervisorAddresses[0], subnetIDs[0])
	hyperCR := srpc.NewClientResource("tcp", hypervisorAddresses[0])
	defer hyperCR.ScheduleClose()
	client, err := hyperCR.GetHTTP(nil, 0)
	if err != nil {
		return err
	}
	defer client.Put()
	hypervisorSubnets, err := hyperclient.ListSubnets(client, false)
	if err != nil {
		return err
	}
	subnetTable := make(map[string]hyper_proto.Subnet, len(hypervisorSubnets))
	for _, subnet := range hypervisorSubnets {
		subnetTable[subnet.Id] = subnet
	}
	var subnets []*hyper_proto.Subnet
	for _, subnetId := range subnetIDs {
		if subnet, ok := subnetTable[subnetId]; !ok {
			return fmt.Errorf("subnet: %s not available on: %s",
				subnetId, hypervisorAddresses[0])
		} else {
			subnets = append(subnets, &subnet)
		}
	}
	info := fm_proto.GetMachineInfoResponse{Subnets: subnets}
	createRequest := hyper_proto.CreateVmRequest{
		DhcpTimeout:      -1,
		EnableNetboot:    true,
		MinimumFreeBytes: uint64(volumeSizes[0]),
		VmInfo: hyper_proto.VmInfo{
			ConsoleType:        hyper_proto.ConsoleVNC,
			Hostname:           "netboot-test",
			MemoryInMiB:        uint64(memory >> 20),
			MilliCPUs:          1000,
			SecondarySubnetIDs: subnetIDs[1:],
			SubnetId:           subnetIDs[0],
		},
	}
	if createRequest.VmInfo.MemoryInMiB < 1 {
		createRequest.VmInfo.MemoryInMiB = 1024
	}
	for _, size := range volumeSizes[1:] {
		createRequest.SecondaryVolumes = append(createRequest.SecondaryVolumes,
			hyper_proto.Volume{Size: uint64(size)})
	}
	var createResponse hyper_proto.CreateVmResponse
	err = hyperclient.CreateVm(client, createRequest, &createResponse, logger)
	if err != nil {
		return err
	}
	err = hyperclient.AcknowledgeVm(client, createResponse.IpAddress)
	if err != nil {
		return err
	}
	logger.Printf("created VM: %s\n", createResponse.IpAddress)
	vmInfo, err := hyperclient.GetVmInfo(client, createResponse.IpAddress)
	if err != nil {
		return err
	}
	vncErrChannel := make(chan error, 1)
	if *vncViewer == "" {
		vncErrChannel <- nil
	} else {
		defer hyperclient.DestroyVm(client, createResponse.IpAddress, nil)
		client, err := srpc.DialHTTP("tcp", hypervisorAddresses[0], 0)
		if err != nil {
			return err
		}
		go func() {
			defer client.Close()
			vncErrChannel <- hyperclient.ConnectToVmConsole(client,
				vmInfo.Address.IpAddress, *vncViewer, logger)
		}()
	}
	info.Machine.NetworkEntry = fm_proto.NetworkEntry{
		Hostname:      vmInfo.Hostname,
		HostIpAddress: vmInfo.Address.IpAddress,
		SubnetId:      subnetIDs[0],
	}
	err = info.Machine.HostMacAddress.UnmarshalText(
		[]byte(vmInfo.Address.MacAddress))
	if err != nil {
		return err
	}
	for index, subnetId := range subnetIDs {
		if index < 1 {
			continue
		}
		address := vmInfo.SecondaryAddresses[index-1]
		var hwAddr fm_proto.HardwareAddr
		if err := hwAddr.UnmarshalText([]byte(address.MacAddress)); err != nil {
			return err
		}
		info.Machine.SecondaryNetworkEntries = append(
			info.Machine.SecondaryNetworkEntries, fm_proto.NetworkEntry{
				HostIpAddress:  address.IpAddress,
				HostMacAddress: hwAddr,
				SubnetId:       subnetId,
			})
	}
	configFiles, err := makeConfigFiles(info, *targetImageName,
		getNetworkEntries(info), false)
	netbootRequest := hyper_proto.NetbootMachineRequest{
		Address:                      vmInfo.Address,
		Files:                        configFiles,
		FilesExpiration:              *netbootFilesTimeout,
		Hostname:                     vmInfo.Hostname,
		NumAcknowledgementsToWaitFor: *numAcknowledgementsToWaitFor,
		OfferExpiration:              *offerTimeout,
		WaitTimeout:                  *netbootTimeout,
	}
	var netbootResponse hyper_proto.NetbootMachineResponse
	err = client.RequestReply("Hypervisor.NetbootMachine", netbootRequest,
		&netbootResponse)
	if err != nil {
		return err
	}
	if err := errors.New(netbootResponse.Error); err != nil {
		return err
	}
	logger.Println("waiting for console exit")
	return <-vncErrChannel
}
