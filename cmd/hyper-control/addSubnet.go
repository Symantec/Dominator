package main

import (
	"fmt"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func addSubnetSubcommand(args []string, logger log.DebugLogger) error {
	err := addSubnet(args[0], args[1], args[2], args[3:], logger)
	if err != nil {
		return fmt.Errorf("Error adding subnet: %s", err)
	}
	return nil
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
	request := proto.UpdateSubnetsRequest{Add: []proto.Subnet{subnet}}
	var reply proto.UpdateSubnetsResponse
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("Hypervisor.UpdateSubnets", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
