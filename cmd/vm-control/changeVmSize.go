package main

import (
	"fmt"
	"net"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func changeVmCPUsSubcommand(args []string, logger log.DebugLogger) error {
	if err := changeVmSize(args[0], 0, *milliCPUs, logger); err != nil {
		return fmt.Errorf("Error changing VM CPUs: %s", err)
	}
	return nil
}

func changeVmMemorySubcommand(args []string, logger log.DebugLogger) error {
	if err := changeVmSize(args[0], uint64(memory), 0, logger); err != nil {
		return fmt.Errorf("Error changing VM memory: %s", err)
	}
	return nil
}

func changeVmSize(vmHostname string, memoryInMiB uint64, milliCPUs uint,
	logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return changeVmSizeOnHypervisor(hypervisor, vmIP, memoryInMiB,
			milliCPUs, logger)
	}
}

func changeVmSizeOnHypervisor(hypervisor string, ipAddr net.IP,
	memoryInMiB uint64, milliCPUs uint, logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.ChangeVmSize(client, proto.ChangeVmSizeRequest{
		IpAddress:   ipAddr,
		MemoryInMiB: memoryInMiB,
		MilliCPUs:   milliCPUs,
	})
}
