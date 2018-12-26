package main

import (
	"fmt"
	"net"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func changeVmDestroyProtectionSubcommand(args []string,
	logger log.DebugLogger) {
	if err := changeVmDestroyProtection(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing VM destroy protection: %s\n",
			err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeVmDestroyProtection(vmHostname string,
	logger log.DebugLogger) error {
	if vmIP, hypervisor, err := lookupVmAndHypervisor(vmHostname); err != nil {
		return err
	} else {
		return changeVmDestroyProtectionOnHypervisor(hypervisor, vmIP, logger)
	}
}

func changeVmDestroyProtectionOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.ChangeVmDestroyProtectionRequest{
		DestroyProtection: *destroyProtection,
		IpAddress:         ipAddr,
	}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.ChangeVmOwnerUsersResponse
	err = client.RequestReply("Hypervisor.ChangeVmDestroyProtection",
		request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
