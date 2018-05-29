package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func probeVmPortSubcommand(args []string, logger log.DebugLogger) {
	if *probePortNum < 1 {
		fmt.Fprintln(os.Stderr, "Must provide -probePortNum flag")
		os.Exit(1)
	}
	if err := probeVmPort(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error probing VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
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
