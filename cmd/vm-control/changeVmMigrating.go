package main

import (
	"fmt"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func setVmMigratingSubcommand(args []string, logger log.DebugLogger) error {
	if err := changeVmMigrationState(args[0], true, logger); err != nil {
		return fmt.Errorf("Error setting VM migration state: %s", err)
	}
	return nil
}

func unsetVmMigratingSubcommand(args []string, logger log.DebugLogger) error {
	if err := changeVmMigrationState(args[0], false, logger); err != nil {
		return fmt.Errorf("Error clearing VM migration state: %s", err)
	}
	return nil
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
