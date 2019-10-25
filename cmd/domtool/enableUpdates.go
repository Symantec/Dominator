package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func enableUpdatesSubcommand(client *srpc.Client, args []string) {
	if err := enableUpdates(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error enabling updates: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
