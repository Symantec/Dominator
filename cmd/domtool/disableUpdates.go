package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func disableUpdatesSubcommand(args []string, logger log.DebugLogger) error {
	if err := disableUpdates(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error disabling updates: %s", err)
	}
	return nil
}

func disableUpdates(client *srpc.Client, reason string) error {
	if reason == "" {
		return errors.New("cannot disable updates: no reason given")
	}
	var request dominator.DisableUpdatesRequest
	var reply dominator.DisableUpdatesResponse
	request.Reason = reason
	return client.RequestReply("Dominator.DisableUpdates", request, &reply)
}
