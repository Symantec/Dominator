package main

import (
	"fmt"
	"net"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func removeIpAddressSubcommand(args []string, logger log.DebugLogger) error {
	ipAddr := net.ParseIP(args[0])
	if len(ipAddr) < 4 {
		return fmt.Errorf("Invalid IP address: %s", args[0])
	}
	err := removeAddress(proto.Address{IpAddress: ipAddr}, logger)
	if err != nil {
		return fmt.Errorf("Error removing IP address: %s", err)
	}
	return nil
}

func removeMacAddressSubcommand(args []string, logger log.DebugLogger) error {
	address := proto.Address{MacAddress: args[0]}
	err := removeAddress(address, logger)
	if err != nil {
		return fmt.Errorf("Error removing MAC address: %s", err)
	}
	return nil
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
