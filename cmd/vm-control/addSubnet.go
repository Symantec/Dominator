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

func addSubnetSubcommand(args []string, logger log.DebugLogger) {
	err := addSubnet(args[0], args[1], args[2], args[3:], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding subnet: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addSubnet(subnetId, ipGateway, ipMask string, nameservers []string,
	logger log.DebugLogger) error {
	nsIPs := make([]net.IP, 0, len(nameservers))
	for _, nameserver := range nameservers {
		nsIPs = append(nsIPs, net.ParseIP(nameserver))
	}
	subnet := proto.Subnet{
		Id:                subnetId,
		IpGateway:         net.ParseIP(ipGateway),
		IpMask:            net.ParseIP(ipMask),
		DomainNameServers: nsIPs,
	}
	subnet.Shrink()
	request := proto.AddSubnetsRequest{Subnets: []proto.Subnet{subnet}}
	var reply proto.AddSubnetsResponse
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("Hypervisor.AddSubnets", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
