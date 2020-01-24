package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func changeTagsSubcommand(args []string, logger log.DebugLogger) error {
	if err := changeTags(logger); err != nil {
		return fmt.Errorf("Error changing Hypervisor tags: %s", err)
	}
	return nil
}

func changeTags(logger log.DebugLogger) error {
	if *hypervisorHostname == "" {
		return errors.New("no hypervisorHostname specified")
	}
	request := proto.ChangeMachineTagsRequest{
		Hostname: *hypervisorHostname,
		Tags:     hypervisorTags,
	}
	var reply proto.ChangeMachineTagsResponse
	clientName := fmt.Sprintf("%s:%d", *fleetManagerHostname,
		*fleetManagerPortNum)
	client, err := srpc.DialHTTPWithDialer("tcp", clientName, rrDialer)
	if err != nil {
		return err
	}
	defer client.Close()
	err = client.RequestReply("FleetManager.ChangeMachineTags", request, &reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}
