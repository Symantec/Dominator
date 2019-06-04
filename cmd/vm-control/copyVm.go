package main

import (
	"errors"
	"fmt"
	"net"

	hyperclient "github.com/Symantec/Dominator/hypervisor/client"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func copyVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := copyVm(args[0], logger); err != nil {
		return fmt.Errorf("Error copying VM: %s", err)
	}
	return nil
}

func copyVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := searchVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return copyVmFromHypervisor(hypervisor, vmIP, logger)
	}
}

func callCopyVm(client *srpc.Client, request hyper_proto.CopyVmRequest,
	reply *hyper_proto.CopyVmResponse, logger log.DebugLogger) error {
	conn, err := client.Call("Hypervisor.CopyVm")
	if err != nil {
		return fmt.Errorf("error calling Hypervisor.CopyVm: %s", err)
	}
	defer conn.Close()
	if err := conn.Encode(request); err != nil {
		return fmt.Errorf("error encoding CopyVm request: %s", err)
	}
	if err := conn.Flush(); err != nil {
		return fmt.Errorf("error flushing CopyVm request: %s", err)
	}
	for {
		var response hyper_proto.CopyVmResponse
		if err := conn.Decode(&response); err != nil {
			return fmt.Errorf("error decoding CopyVm response: %s", err)
		}
		if response.Error != "" {
			return errors.New(response.Error)
		}
		if response.ProgressMessage != "" {
			logger.Debugln(0, response.ProgressMessage)
		}
		if response.Final {
			*reply = response
			return nil
		}
	}
}

func copyVmFromHypervisor(sourceHypervisorAddress string, vmIP net.IP,
	logger log.DebugLogger) error {
	destHypervisorAddress, err := getHypervisorAddress()
	if err != nil {
		return err
	}
	sourceHypervisor, err := dialHypervisor(sourceHypervisorAddress)
	if err != nil {
		return err
	}
	defer sourceHypervisor.Close()
	sourceVmInfo, err := hyperclient.GetVmInfo(sourceHypervisor, vmIP)
	if err != nil {
		return err
	}
	vmInfo := createVmInfoFromFlags()
	vmInfo.DestroyProtection = vmInfo.DestroyProtection ||
		sourceVmInfo.DestroyProtection
	if vmInfo.Hostname == "" {
		vmInfo.Hostname = sourceVmInfo.Hostname
	}
	if vmInfo.MemoryInMiB < 1 {
		vmInfo.MemoryInMiB = sourceVmInfo.MemoryInMiB
	}
	if vmInfo.MilliCPUs < 1 {
		vmInfo.MilliCPUs = sourceVmInfo.MilliCPUs
	}
	if len(vmInfo.OwnerGroups) < 1 {
		vmInfo.OwnerGroups = sourceVmInfo.OwnerGroups
	}
	if len(vmInfo.OwnerUsers) < 1 {
		vmInfo.OwnerUsers = sourceVmInfo.OwnerUsers
	}
	if len(vmInfo.Tags) < 1 {
		vmInfo.Tags = sourceVmInfo.Tags
	}
	if len(vmInfo.SecondarySubnetIDs) < 1 {
		vmInfo.SecondarySubnetIDs = sourceVmInfo.SecondarySubnetIDs
	}
	if vmInfo.SubnetId == "" {
		vmInfo.SubnetId = sourceVmInfo.SubnetId
	}
	accessToken, err := getVmAccessTokenClient(sourceHypervisor, vmIP)
	if err != nil {
		return err
	}
	defer discardAccessToken(sourceHypervisor, vmIP)
	destHypervisor, err := dialHypervisor(destHypervisorAddress)
	if err != nil {
		return err
	}
	defer destHypervisor.Close()
	request := hyper_proto.CopyVmRequest{
		AccessToken:      accessToken,
		IpAddress:        vmIP,
		SourceHypervisor: sourceHypervisorAddress,
		VmInfo:           vmInfo,
	}
	var reply hyper_proto.CopyVmResponse
	logger.Debugf(0, "copying VM to %s\n", destHypervisorAddress)
	if err := callCopyVm(destHypervisor, request, &reply, logger); err != nil {
		return err
	}
	err = hyperclient.AcknowledgeVm(destHypervisor, reply.IpAddress)
	if err != nil {
		return fmt.Errorf("error acknowledging VM: %s", err)
	}
	fmt.Println(reply.IpAddress)
	return nil
}
