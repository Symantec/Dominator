package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func restoreVmUserDataSubcommand(args []string, logger log.DebugLogger) {
	if err := restoreVmUserData(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error restoring VM user data: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func restoreVmUserData(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return restoreVmUserDataOnHypervisor(hypervisor, vmIP, logger)
	}
}

func restoreVmUserDataOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.RestoreVmUserDataRequest{ipAddr}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.RestoreVmUserDataResponse
	err = client.RequestReply("Hypervisor.RestoreVmUserData", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
