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

func discardVmOldUserDataSubcommand(args []string, logger log.DebugLogger) {
	if err := discardVmOldUserData(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error discarding VM old user data: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func discardVmOldUserData(ipAddr string, logger log.DebugLogger) error {
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return discardVmOldUserDataOnHypervisor(hypervisor, net.ParseIP(ipAddr),
		logger)
}

func discardVmOldUserDataOnHypervisor(hypervisor string, ipAddr net.IP,
	logger log.DebugLogger) error {
	request := proto.DiscardVmOldUserDataRequest{ipAddr}
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.DiscardVmOldUserDataResponse
	err = client.RequestReply("Hypervisor.DiscardVmOldUserData", request,
		&reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
