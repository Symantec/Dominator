package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func getVmInfoSubcommand(args []string, logger log.DebugLogger) error {
	if err := getVmInfo(args[0], logger); err != nil {
		return fmt.Errorf("Error getting VM info: %s", err)
	}
	return nil
}

func getVmInfo(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return getVmInfoOnHypervisor(hypervisor, vmIP, logger)
	}
}

func getVmInfoOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	if vmInfo, err := getVmInfoClient(client, ipAddr); err != nil {
		return err
	} else {
		return json.WriteWithIndent(os.Stdout, "    ", vmInfo)
	}
}

func getVmInfoClient(client *srpc.Client, ipAddr net.IP) (proto.VmInfo, error) {
	request := proto.GetVmInfoRequest{ipAddr}
	var reply proto.GetVmInfoResponse
	err := client.RequestReply("Hypervisor.GetVmInfo", request, &reply)
	if err != nil {
		return proto.VmInfo{}, err
	}
	if err := errors.New(reply.Error); err != nil {
		return proto.VmInfo{}, err
	}
	return reply.VmInfo, nil
}
