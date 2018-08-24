package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func changeVmTagsSubcommand(args []string, logger log.DebugLogger) {
	if err := changeVmTags(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing VM tags: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeVmTags(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return changeVmTagsOnHypervisor(hypervisor, vmIP, logger)
	}
}

func changeVmTagsOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ChangeVmTagsRequest{ipAddr, vmTags}
	client, err := dialHypervisor(hypervisor)
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
