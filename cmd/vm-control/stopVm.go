package main

import (
	"fmt"
	"net"
	"os"

	hyperclient "github.com/Symantec/Dominator/hypervisor/client"
	"github.com/Symantec/Dominator/lib/log"
)

func stopVmSubcommand(args []string, logger log.DebugLogger) {
	if err := stopVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func stopVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return stopVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func stopVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.StopVm(client, ipAddr, nil)
}
