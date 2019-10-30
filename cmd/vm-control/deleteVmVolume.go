package main

import (
	"fmt"
	"net"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func deleteVmVolumeSubcommand(args []string, logger log.DebugLogger) error {
	if err := deleteVmVolume(args[0], logger); err != nil {
		return fmt.Errorf("Error deleting VM volume: %s", err)
	}
	return nil
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
