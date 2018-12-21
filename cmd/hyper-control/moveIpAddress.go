package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func moveIpAddressSubcommand(args []string, logger log.DebugLogger) {
	if err := moveIpAddress(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error moving IP address: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
	client, err := srpc.DialHTTP("tcp", clientName, 0)
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
