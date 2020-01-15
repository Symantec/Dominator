package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func enableUpdatesSubcommand(args []string, logger log.DebugLogger) error {
	if err := enableUpdates(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error enabling updates: %s", err)
	}
	return nil
}

func enableUpdates(client *srpc.Client, reason string) error {
	if reason == "" {
		return errors.New("cannot enable updates: no reason given")
	}
	var request dominator.EnableUpdatesRequest
	var reply dominator.EnableUpdatesResponse
	request.Reason = reason
	return client.RequestReply("Dominator.EnableUpdates", request, &reply)
}
