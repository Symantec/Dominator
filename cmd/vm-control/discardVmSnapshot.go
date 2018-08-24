package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func discardVmSnapshotSubcommand(args []string, logger log.DebugLogger) {
	if err := discardVmSnapshot(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error discarding VM snapshot: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func discardVmSnapshot(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return discardVmSnapshotOnHypervisor(hypervisor, vmIP, logger)
	}
}

func discardVmSnapshotOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.DiscardVmSnapshotRequest{ipAddr}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.DiscardVmSnapshotResponse
	err = client.RequestReply("Hypervisor.DiscardVmSnapshot", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
