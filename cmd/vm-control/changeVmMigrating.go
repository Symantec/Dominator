package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func setVmMigratingSubcommand(args []string, logger log.DebugLogger) {
	if err := changeVmMigrationState(args[0], true, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting VM migration state: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func unsetVmMigratingSubcommand(args []string, logger log.DebugLogger) {
	if err := changeVmMigrationState(args[0], false, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing VM migration state: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func changeVmMigrationState(vmHostname string, enable bool,
	logger log.DebugLogger) error {
	ipAddr, err := lookupIP(vmHostname)
	if err != nil {
		return err
	}
	var hypervisor string
	if *hypervisorHostname != "" {
		hypervisor = fmt.Sprintf("%s:%d",
			*hypervisorHostname, *hypervisorPortNum)
	} else {
		hypervisor = fmt.Sprintf("localhost:%d", *hypervisorPortNum)
	}
	request := proto.PrepareVmForMigrationRequest{
		Enable:    enable,
		IpAddress: ipAddr}
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	var reply proto.PrepareVmForMigrationResponse
	err = client.RequestReply("Hypervisor.PrepareVmForMigration", request,
		&reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
