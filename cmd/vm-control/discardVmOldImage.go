package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func discardVmOldImageSubcommand(args []string, logger log.DebugLogger) error {
	if err := discardVmOldImage(args[0], logger); err != nil {
		return fmt.Errorf("Error discarding VM old image: %s", err)
	}
	return nil
}

func discardVmOldImage(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return discardVmOldImageOnHypervisor(hypervisor, vmIP, logger)
	}
}

func discardVmOldImageOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.DiscardVmOldImageRequest{ipAddr}
	client, err := dialHypervisor(hypervisor)
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
