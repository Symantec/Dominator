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

func removeIpAddressSubcommand(args []string, logger log.DebugLogger) {
	ipAddr := net.ParseIP(args[0])
	if len(ipAddr) < 4 {
		fmt.Fprintf(os.Stderr, "Invalid IP address: %s\n", args[0])
		os.Exit(1)
	}
	err := removeAddress(proto.Address{IpAddress: ipAddr}, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing IP address: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func removeMacAddressSubcommand(args []string, logger log.DebugLogger) {
	address := proto.Address{MacAddress: args[0]}
	err := removeAddress(address, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error removing MAC address: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func removeAddress(address proto.Address, logger log.DebugLogger) error {
	address.Shrink()
	request := proto.ChangeAddressPoolRequest{
		AddressesToRemove: []proto.Address{address}}
	var reply proto.ChangeAddressPoolResponse
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("Hypervisor.ChangeAddressPool", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
