package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

func getUpdatesSubcommand(args []string, logger log.DebugLogger) error {
	if err := getUpdates(logger); err != nil {
		return fmt.Errorf("Error getting updates: %s", err)
	}
	return nil
}

func getUpdates(logger log.DebugLogger) error {
	if *hypervisorHostname != "" {
		return getUpdatesOnHypervisor(
			fmt.Sprintf("%s:%d", *hypervisorHostname, *hypervisorPortNum),
			logger)
	} else if *fleetManagerHostname != "" {
		return getUpdatesOnFleetManager(
			fmt.Sprintf("%s:%d", *fleetManagerHostname, *fleetManagerPortNum),
			logger)
	} else {
		return getUpdatesOnHypervisor(fmt.Sprintf(":%d", *hypervisorPortNum),
			logger)
	}

}

func getUpdatesOnFleetManager(fleetManager string,
	logger log.DebugLogger) error {
	client, err := srpc.DialHTTPWithDialer("tcp", fleetManager, rrDialer)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("FleetManager.GetUpdates")
	if err != nil {
		return err
	}
	defer conn.Close()
	request := fm_proto.GetUpdatesRequest{Location: *location}
	if err := conn.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	for {
		var update fm_proto.Update
		if err := conn.Decode(&update); err != nil {
			return err
		}
		if err := errors.New(update.Error); err != nil {
			return err
		}
		if err := json.WriteWithIndent(os.Stdout, "    ", update); err != nil {
			return err
		}
	}
}

func getUpdatesOnHypervisor(hypervisor string, logger log.DebugLogger) error {
	client, err := srpc.DialHTTP("tcp", hypervisor, 0)
	if err != nil {
		return err
	}
	defer client.Close()
	conn, err := client.Call("Hypervisor.GetUpdates")
	if err != nil {
		return err
	}
	defer conn.Close()
	for {
		var update hyper_proto.Update
		if err := conn.Decode(&update); err != nil {
			return err
		}
		if err := json.WriteWithIndent(os.Stdout, "    ", update); err != nil {
			return err
		}
	}
}
