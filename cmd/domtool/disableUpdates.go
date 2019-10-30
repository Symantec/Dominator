package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func disableUpdatesSubcommand(client *srpc.Client, args []string) {
	if err := disableUpdates(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error disabling updates: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
