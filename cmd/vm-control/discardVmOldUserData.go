package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func discardVmOldUserDataSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := discardVmOldUserData(args[0], logger); err != nil {
		return fmt.Errorf("Error discarding VM old user data: %s", err)
	}
	return nil
}

func discardVmOldUserData(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return discardVmOldUserDataOnHypervisor(hypervisor, vmIP, logger)
	}
}

func discardVmOldUserDataOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.DiscardVmOldUserDataRequest{ipAddr}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.DiscardVmOldUserDataResponse
	err = client.RequestReply("Hypervisor.DiscardVmOldUserData", request,
		&reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
