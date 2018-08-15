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
	var ipAddr string
	if len(args) > 1 {
		ipAddr = args[1]
	}
	err := addAddress(args[0], ipAddr, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding address: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addAddress(macAddr, ipAddr string, logger log.DebugLogger) error {
	address := proto.Address{MacAddress: macAddr}
	if ipAddr != "" {
		address.IpAddress = net.ParseIP(ipAddr)
	}
	address.Shrink()
	request := proto.ChangeAddressPoolRequest{
		AddressesToAdd: []proto.Address{address}}
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
