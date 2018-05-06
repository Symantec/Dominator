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

func discardVmOldImageSubcommand(args []string, logger log.DebugLogger) {
	if err := discardVmOldImage(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error discarding VM old image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func discardVmOldImage(ipAddr string, logger log.DebugLogger) error {
	vmIP := net.ParseIP(ipAddr)
	if hypervisor, err := findHypervisor(vmIP); err != nil {
		return err
	} else {
		return discardVmOldImageOnHypervisor(hypervisor, vmIP, logger)
	}
}

func discardVmOldImageOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.DiscardVmOldImageRequest{ipAddr}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.DiscardVmOldImageResponse
	err = client.RequestReply("Hypervisor.DiscardVmOldImage", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
