package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func moveIpAddressSubcommand(args []string, logger log.DebugLogger) error {
	if err := moveIpAddress(args[0], logger); err != nil {
		return fmt.Errorf("Error moving IP address: %s", err)
	}
	return nil
}

func moveIpAddress(addr string, logger log.DebugLogger) error {
	ipAddr := net.ParseIP(addr)
	if len(ipAddr) < 4 {
		return fmt.Errorf("invalid IP address: %s", addr)
	}
	request := proto.MoveIpAddressesRequest{
		HypervisorHostname: *hypervisorHostname,
		IpAddresses:        []net.IP{ipAddr},
	}
	var reply proto.MoveIpAddressesResponse
	clientName := fmt.Sprintf("%s:%d",
		*fleetManagerHostname, *fleetManagerPortNum)
	client, err := srpc.DialHTTPWithDialer("tcp", clientName, rrDialer)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("FleetManager.MoveIpAddresses", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
