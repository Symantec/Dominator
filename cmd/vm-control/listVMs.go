package main

import (
	"encoding/gob"
	"fmt"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/verstr"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func listVMsSubcommand(args []string, logger log.DebugLogger) error {
	if err := listVMs(logger); err != nil {
		return fmt.Errorf("Error listing VMs: %s", err)
	}
	return nil
}

func listVMs(logger log.DebugLogger) error {
	if *fleetManagerHostname != "" {
		fleetManager := fmt.Sprintf("%s:%d",
			*fleetManagerHostname, *fleetManagerPortNum)
		return listVMsByLocation(fleetManager, *location, logger)
	}
	hypervisor := fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum)
	return listVMsOnHypervisor(hypervisor, logger)
}

func listVMsByLocation(fleetManager string, location string,
	logger log.DebugLogger) error {
	client, err := dialFleetManager(fleetManager)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("FleetManager.ListVMsInLocation")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	decoder := gob.NewDecoder(conn)
	request := fm_proto.ListVMsInLocationRequest{location}
	if err := encoder.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	var addresses []string
	for {
		var reply fm_proto.ListVMsInLocationResponse
		if err := decoder.Decode(&reply); err != nil {
			return err
		}
		if err := errors.New(reply.Error); err != nil {
			return err
		}
		if len(reply.IpAddresses) < 1 {
			break
		}
		for _, ipAddress := range reply.IpAddresses {
			addresses = append(addresses, ipAddress.String())
		}
	}
	verstr.Sort(addresses)
	for _, address := range addresses {
		if _, err := fmt.Println(address); err != nil {
			return err
		}
	}
	return nil
}

func listVMsOnHypervisor(hypervisor string, logger log.DebugLogger) error {
	client, err := dialHypervisor(hypervisor)
	if err != nil {
		return err
	}
	defer client.Close()
	request := hyper_proto.ListVMsRequest{
		OwnerUsers: ownerUsers,
		Sort:       true,
	}
	var reply hyper_proto.ListVMsResponse
	err = client.RequestReply("Hypervisor.ListVMs", request, &reply)
	if err != nil {
		return err
	}
	for _, ipAddress := range reply.IpAddresses {
		if _, err := fmt.Println(ipAddress); err != nil {
			return err
		}
	}
	return nil
}
