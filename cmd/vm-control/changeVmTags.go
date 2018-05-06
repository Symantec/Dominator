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

func changeVmTagsSubcommand(args []string, logger log.DebugLogger) {
	if err := changeVmTags(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing VM tags: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeVmTags(ipAddr string, logger log.DebugLogger) error {
	vmIP := net.ParseIP(ipAddr)
	if hypervisor, err := findHypervisor(vmIP); err != nil {
		return err
	} else {
		return changeVmTagsOnHypervisor(hypervisor, vmIP, logger)
	}
}

func changeVmTagsOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ChangeVmTagsRequest{ipAddr, vmTags}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.ChangeVmTagsResponse
	err = client.RequestReply("Hypervisor.ChangeVmTags", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
