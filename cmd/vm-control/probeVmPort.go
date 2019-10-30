package main

import (
	"fmt"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func probeVmPortSubcommand(args []string, logger log.DebugLogger) error {
	if *probePortNum < 1 {
		return fmt.Errorf("Must provide -probePortNum flag")
	}
	if err := probeVmPort(args[0], logger); err != nil {
		return fmt.Errorf("Error probing VM: %s", err)
	}
	return nil
}

func probeVmPort(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return probeVmPortOnHypervisor(hypervisor, vmIP, logger)
	}
}

func probeVmPortOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	return probeVmPortOnHypervisorClient(client, ipAddr, logger)
}

func probeVmPortOnHypervisorClient(client *srpc.Client, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ProbeVmPortRequest{
		IpAddress:  ipAddr,
		PortNumber: *probePortNum,
		Timeout:    *probeTimeout,
	}
	var reply proto.ProbeVmPortResponse
	err := client.RequestReply("Hypervisor.ProbeVmPort", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	if !reply.PortIsOpen {
		return errors.New("Timed out probing port")
	}
	logger.Debugf(0, "Port %d is open\n", *probePortNum)
	return nil
}
