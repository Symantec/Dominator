package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func becomePrimaryVmOwnerSubcommand(args []string, logger log.DebugLogger) {
	if err := becomePrimaryVmOwner(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error becoming primary VM owner: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func becomePrimaryVmOwner(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return becomePrimaryVmOwnerOnHypervisor(hypervisor, vmIP, logger)
	}
}

func becomePrimaryVmOwnerOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.BecomePrimaryVmOwnerRequest{ipAddr}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.BecomePrimaryVmOwnerResponse
	err = client.RequestReply("Hypervisor.BecomePrimaryVmOwner", request,
		&reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
