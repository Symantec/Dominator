package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func restoreVmFromSnapshotSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := restoreVmFromSnapshot(args[0], logger); err != nil {
		return fmt.Errorf("Error restoring VM from snapshot: %s", err)
	}
	return nil
}

func restoreVmFromSnapshot(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return restoreVmFromSnapshotOnHypervisor(hypervisor, vmIP, logger)
	}
}

func restoreVmFromSnapshotOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.RestoreVmFromSnapshotRequest{ipAddr, *forceIfNotStopped}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.RestoreVmFromSnapshotResponse
	err = client.RequestReply("Hypervisor.RestoreVmFromSnapshot", request,
		&reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
