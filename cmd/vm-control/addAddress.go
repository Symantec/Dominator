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

func addAddressSubcommand(args []string, logger log.DebugLogger) {
	err := addAddress(args[0], args[1], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding address: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addAddress(macAddr, ipAddr string, logger log.DebugLogger) error {
	address := proto.Address{
		IpAddress:  net.ParseIP(ipAddr),
		MacAddress: macAddr,
	}
	address.Shrink()
	request := proto.AddAddressesToPoolRequest{
		Addresses: []proto.Address{address}}
	var reply proto.AddAddressesToPoolResponse
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("Hypervisor.AddAddressesToPool", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
