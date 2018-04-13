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

func restoreVmImageSubcommand(args []string, logger log.DebugLogger) {
	if err := restoreVmImage(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error restoring VM image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func restoreVmImage(ipAddr string, logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return restoreVmImageOnHypervisor(hypervisor, net.ParseIP(ipAddr), logger)
}

func restoreVmImageOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.RestoreVmImageRequest{ipAddr}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.RestoreVmImageResponse
	err = client.RequestReply("Hypervisor.RestoreVmImage", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
