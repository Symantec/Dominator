package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func destroyVmSubcommand(args []string, logger log.DebugLogger) {
	if err := destroyVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error destroying VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func destroyVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return destroyVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func destroyVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return destroyVmOnHypervisorClient(client, ipAddr)
}

func destroyVmOnHypervisorClient(client *srpc.Client, ipAddr net.IP) error {
	request := proto.DestroyVmRequest{ipAddr}
	var reply proto.DestroyVmResponse
	err := client.RequestReply("Hypervisor.DestroyVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
