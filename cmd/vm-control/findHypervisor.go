package main

import (
	"fmt"
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func findHypervisor(vmIpAddr net.IP) (string, error) {
	if *fleetManagerHostname != "" {
		cm := fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum)
		client, err := srpc.DialHTTP("tcp", cm, time.Second*10)
		if err != nil {
			return "", err
		}
		defer client.Close()
		return findHypervisorClient(client, vmIpAddr)
	} else if *hypervisorHostname != "" {
		return fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum),
			nil
	} else {
		return fmt.Sprintf("localhost:%d", *hypervisorPortNum), nil
	}
}

func findHypervisorClient(client *srpc.Client,
	vmIpAddr net.IP) (string, error) {
	request := proto.GetHypervisorForVMRequest{vmIpAddr}
	var reply proto.GetHypervisorForVMResponse
	err := client.RequestReply("FleetManager.GetHypervisorForVM", request,
		&reply)
	if err != nil {
		return "", err
	}
	if err := errors.New(reply.Error); err != nil {
		return "", err
	}
	return reply.HypervisorAddress, nil
}

func lookupIP(vmHostname string) (net.IP, error) {
	if ips, err := net.LookupIP(vmHostname); err != nil {
		return nil, err
	} else if len(ips) != 1 {
		return nil, fmt.Errorf("num IPs: %d != 1", len(ips))
	} else {
		return ips[0], nil
	}
}

func lookupVmAndHypervisor(vmHostname string) (net.IP, string, error) {
	if vmIpAddr, err := lookupIP(vmHostname); err != nil {
		return nil, "", err
	} else if hypervisor, err := findHypervisor(vmIpAddr); err != nil {
		return nil, "", err
	} else {
		return vmIpAddr, hypervisor, nil
	}
}
