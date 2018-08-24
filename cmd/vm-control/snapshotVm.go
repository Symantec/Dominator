package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func snapshotVmSubcommand(args []string, logger log.DebugLogger) {
	if err := snapshotVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error snapshotting VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func snapshotVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return snapshotVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func snapshotVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.SnapshotVmRequest{ipAddr, *forceIfNotStopped,
		*snapshotRootOnly}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.SnapshotVmResponse
	err = client.RequestReply("Hypervisor.SnapshotVm", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
