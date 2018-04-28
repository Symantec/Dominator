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

func changeVmOwnerUsersSubcommand(args []string, logger log.DebugLogger) {
	if err := changeVmOwnerUsers(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing VM owner users: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeVmOwnerUsers(ipAddr string, logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return changeVmOwnerUsersOnHypervisor(hypervisor, net.ParseIP(ipAddr),
		logger)
}

func changeVmOwnerUsersOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ChangeVmOwnerUsersRequest{ipAddr, ownerUsers}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.ChangeVmOwnerUsersResponse
	err = client.RequestReply("Hypervisor.ChangeVmOwnerUsers", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
