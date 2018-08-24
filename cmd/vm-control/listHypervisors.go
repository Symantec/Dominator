package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func listHypervisorsSubcommand(args []string, logger log.DebugLogger) {
	if err := listHypervisors(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing Hypervisors: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listHypervisors(logger log.DebugLogger) error {
	fleetManager := fmt.Sprintf("%s:%d",
		*fleetManagerHostname, *fleetManagerPortNum)
	client, err := dialFleetManager(fleetManager)
	if err != nil {
		return err
	}
	defer client.Close()
	request := proto.ListHypervisorsInLocationRequest{
		Location: *location,
		SubnetId: *subnetId,
	}
	var reply proto.ListHypervisorsInLocationResponse
	err = client.RequestReply("FleetManager.ListHypervisorsInLocation",
		request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	for _, address := range reply.HypervisorAddresses {
		hypervisor := strings.Split(address, ":")[0]
		if _, err := fmt.Println(hypervisor); err != nil {
			return err
		}
	}
	return nil
}
