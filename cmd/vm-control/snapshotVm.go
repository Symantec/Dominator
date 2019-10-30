package main

import (
	"fmt"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func snapshotVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := snapshotVm(args[0], logger); err != nil {
		return fmt.Errorf("Error snapshotting VM: %s", err)
	}
	return nil
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
