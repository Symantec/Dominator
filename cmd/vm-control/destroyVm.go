package main

import (
	"fmt"
	"net"
	"os"

	hyperclient "github.com/Symantec/Dominator/hypervisor/client"
	"github.com/Symantec/Dominator/lib/log"
)

func destroyVmSubcommand(args []string, logger log.DebugLogger) {
	if err := destroyVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error destroying VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func destroyVm(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return destroyVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func destroyVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.DestroyVm(client, ipAddr, nil)
}
