package main

import (
	"fmt"
	"os"

	fmclient "github.com/Cloud-Foundations/Dominator/fleetmanager/client"
	hyperclient "github.com/Cloud-Foundations/Dominator/hypervisor/client"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func powerOffSubcommand(args []string, logger log.DebugLogger) error {
	err := powerOff(logger)
	if err != nil {
		return fmt.Errorf("Error powering down: %s", err)
	}
	return nil
}

func powerOff(logger log.DebugLogger) error {
	if *hypervisorHostname == "" {
		return errors.New("cannot power down myself")
	}
	if hostname, err := os.Hostname(); err != nil {
		return err
	} else if *hypervisorHostname == hostname {
		return errors.New("cannot power down myself")
	}
	clientName := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	return hyperclient.PowerOff(client, false)
}

func powerOnSubcommand(args []string, logger log.DebugLogger) error {
	err := powerOn(logger)
	if err != nil {
		return fmt.Errorf("Error powering up: %s", err)
	}
	return nil
}

func powerOn(logger log.DebugLogger) error {
	if *hypervisorHostname == "" {
		return errors.New("unspecified Hypervisor")
	}
	if *fleetManagerHostname == "" {
		return errors.New("unspecified Fleet Manager")
	}
	clientName := fmt.Sprintf("%s:%d", *fleetManagerHostname,
		*fleetManagerPortNum)
	client, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	return fmclient.PowerOnMachine(client, *hypervisorHostname)
}
