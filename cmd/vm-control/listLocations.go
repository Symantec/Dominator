package main

import (
	"fmt"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func listLocationsSubcommand(args []string, logger log.DebugLogger) error {
	var topLocation string
	if len(args) > 0 {
		topLocation = args[0]
	}
	if err := listLocations(topLocation, logger); err != nil {
		return fmt.Errorf("Error listing locations: %s", err)
	}
	return nil
}

func listLocations(topLocation string, logger log.DebugLogger) error {
	fleetManager := fmt.Sprintf("%s:%d",
		*fleetManagerHostname, *fleetManagerPortNum)
	client, err := dialFleetManager(fleetManager)
	if err != nil {
		return err
	}
	defer client.Close()
	request := proto.ListHypervisorLocationsRequest{topLocation}
	var reply proto.ListHypervisorLocationsResponse
	err = client.RequestReply("FleetManager.ListHypervisorLocations",
		request, &reply)
	if err != nil {
		return err
	}
	if err := errors.New(reply.Error); err != nil {
		return err
	}
	for _, location := range reply.Locations {
		if _, err := fmt.Println(location); err != nil {
			return err
		}
	}
	return nil
}
