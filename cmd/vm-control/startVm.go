package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func startVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := startVm(args[0], logger); err != nil {
		return fmt.Errorf("Error starting VM: %s", err)
	}
	return nil
}

func startVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return startVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func startVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.StartVmRequest{
		DhcpTimeout: *dhcpTimeout,
		IpAddress:   ipAddr,
	}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.StartVmResponse
	err = client.RequestReply("Hypervisor.StartVm", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	if reply.DhcpTimedOut {
		return errors.New("DHCP ACK timed out")
	}
	return maybeWatchVm(client, hypervisor, ipAddr, logger)
}
