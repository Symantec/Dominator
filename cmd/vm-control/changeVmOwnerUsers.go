package main

import (
	"fmt"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func changeVmOwnerUsersSubcommand(args []string, logger log.DebugLogger) error {
	if err := changeVmOwnerUsers(args[0], logger); err != nil {
		return fmt.Errorf("Error changing VM owner users: %s", err)
	}
	return nil
}

func changeVmOwnerUsers(vmHostname string, logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return changeVmOwnerUsersOnHypervisor(hypervisor, vmIP, logger)
	}
}

func changeVmOwnerUsersOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ChangeVmOwnerUsersRequest{ipAddr, ownerUsers}
	client, err := dialHypervisor(hypervisor)
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
