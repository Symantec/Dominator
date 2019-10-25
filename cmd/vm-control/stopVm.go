package main

import (
	"fmt"
	"net"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func stopVmSubcommand(args []string, logger log.DebugLogger) error {
	if err := stopVm(args[0], logger); err != nil {
		return fmt.Errorf("Error stopping VM: %s", err)
	}
	return nil
}

func logVmName(client *srpc.Client, ipAddr net.IP, action string,
	logger log.DebugLogger) {
	if vmInfo, err := hyperclient.GetVmInfo(client, ipAddr); err != nil {
		return
	} else {
		name := vmInfo.Hostname
		if name == "" {
			name = vmInfo.Tags["Name"]
		}
		if name == "" {
			return
		}
		logger.Debugf(0, "%s %s\n", action, name)
	}
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
	logVmName(client, ipAddr, "stopping", logger)
	return hyperclient.StopVm(client, ipAddr, nil)
}
