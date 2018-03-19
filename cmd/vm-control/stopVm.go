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

func stopVmSubcommand(args []string, logger log.DebugLogger) {
	if err := stopVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func stopVm(ipAddr string, logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return stopVmOnHypervisor(hypervisor, net.ParseIP(ipAddr), logger)
}

func stopVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.StopVmRequest{ipAddr}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.StopVmResponse
	err = client.RequestReply("Hypervisor.StopVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
