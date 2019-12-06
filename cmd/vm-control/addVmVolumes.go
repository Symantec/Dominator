package main

import (
	"fmt"
	"net"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func addVmVolumesSubcommand(args []string, logger log.DebugLogger) error {
	if err := addVmVolumes(args[0], logger); err != nil {
		return fmt.Errorf("Error adding VM volumes: %s", err)
	}
	return nil
}

func addVmVolumes(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return addVmVolumesOnHypervisor(hypervisor, vmIP, logger)
	}
}

func addVmVolumesOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	sizes := make([]uint64, 0, len(secondaryVolumeSizes))
	for _, size := range secondaryVolumeSizes {
		sizes = append(sizes, uint64(size))
	}
	return hyperclient.AddVmVolumes(client, ipAddr, sizes)
}
