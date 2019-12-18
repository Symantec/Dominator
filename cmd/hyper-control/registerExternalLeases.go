package main

import (
	"fmt"

	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func registerExternalLeasesSubcommand(args []string,
	logger log.DebugLogger) error {
	err := registerExternalLeases(logger)
	if err != nil {
		return fmt.Errorf("error registering external leases: %s", err)
	}
	return nil
}

func registerExternalLeases(logger log.DebugLogger) error {
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.RegisterExternalLeases(client, externalLeaseAddresses,
		externalLeaseHostnames)
}
