package main

import (
	"fmt"
	"net"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func connectToVmConsoleSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := connectToVmConsole(args[0], logger); err != nil {
		return fmt.Errorf("Error connecting to VM console: %s", err)
	}
	return nil
}

func connectToVmConsole(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return connectToVmConsoleOnHypervisor(hypervisor, vmIP, logger)
	}
}

func connectToVmConsoleOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.ConnectToVmConsole(client, ipAddr, *vncViewer, logger)
}
