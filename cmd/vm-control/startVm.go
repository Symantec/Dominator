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

func startVmSubcommand(args []string, logger log.DebugLogger) {
	if err := startVm(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting VM: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func startVm(ipAddr string, logger log.DebugLogger) error {
	vmIP := net.ParseIP(ipAddr)
	if hypervisor, err := findHypervisor(vmIP); err != nil {
		return err
	} else {
		return startVmOnHypervisor(hypervisor, vmIP, logger)
	}
}

func startVmOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.StartVmRequest{
		DhcpTimeout: *dhcpTimeout,
		IpAddress:   ipAddr,
	}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.StartVmResponse
	err = client.RequestReply("Hypervisor.StartVm", request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	if reply.DhcpTimedOut {
		return errors.New("DHCP ACK timed out")
	}
	return maybeWatchVm(client, hypervisor, ipAddr, logger)
}
