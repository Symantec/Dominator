package main

import (
	"fmt"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func changeVmConsoleTypeSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := changeVmConsoleType(args[0], logger); err != nil {
		return fmt.Errorf("Error changing VM console type: %s", err)
	}
	return nil
}

func changeVmConsoleType(vmHostname string,
	logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return changeVmConsoleTypeOnHypervisor(hypervisor, vmIP, logger)
	}
}

func changeVmConsoleTypeOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ChangeVmConsoleTypeRequest{
		ConsoleType: consoleType,
		IpAddress:   ipAddr,
	}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.ChangeVmOwnerUsersResponse
	err = client.RequestReply("Hypervisor.ChangeVmConsoleType",
		request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
