package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func listVMsSubcommand(args []string, logger log.DebugLogger) {
	if err := listVMs(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing VMs: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listVMs(logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return listVMsOnHypervisor(hypervisor, logger)
}

func listVMsOnHypervisor(hypervisor string, logger log.DebugLogger) error {
	request := proto.ListVMsRequest{true}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.ListVMsResponse
	err = client.RequestReply("Hypervisor.ListVMs", request, &reply)
	if err != nil {
		return err
	}
	for _, ipAddress := range reply.IpAddresses {
		if _, err := fmt.Println(ipAddress); err != nil {
			return err
		}
	}
	return nil
}
