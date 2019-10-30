package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func listHypervisorsSubcommand(args []string, logger log.DebugLogger) error {
	if err := listHypervisors(logger); err != nil {
		return fmt.Errorf("Error listing Hypervisors: %s", err)
	}
	return nil
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
		IncludeUnhealthy: *includeUnhealthy,
		Location:         *location,
		SubnetId:         *subnetId,
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
	hypervisors := make([]string, 0, len(reply.HypervisorAddresses))
	for _, address := range reply.HypervisorAddresses {
		hypervisors = append(hypervisors, strings.Split(address, ":")[0])
	}
	sort.Strings(hypervisors)
	for _, hypervisor := range hypervisors {
		if _, err := fmt.Println(hypervisor); err != nil {
			return err
		}
	}
	return nil
}
