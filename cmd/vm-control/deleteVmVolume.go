package main

import (
	"fmt"
	"net"
	"os"

	hyperclient "github.com/Symantec/Dominator/hypervisor/client"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
)

func deleteVmVolumeSubcommand(args []string, logger log.DebugLogger) {
	if err := deleteVmVolume(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting VM volume: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func deleteVmVolume(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return deleteVmVolumeOnHypervisor(hypervisor, vmIP, logger)
	}
}

func deleteVmVolumeOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	if *volumeIndex < 1 {
		return errors.New("cannot delete root volume")
	}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.DeleteVmVolume(client, ipAddr, nil, *volumeIndex)
}
