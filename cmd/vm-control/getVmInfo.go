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

func getVmInfoSubcommand(args []string, logger log.DebugLogger) {
	if err := getVmInfo(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting VM info: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
	request := proto.GetVmInfoRequest{ipAddr}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.GetVmInfoResponse
	err = client.RequestReply("Hypervisor.GetVmInfo", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	return json.WriteWithIndent(os.Stdout, "    ", reply.VmInfo)
}
