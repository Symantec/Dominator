package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/dominator"
	"os"
)

func clearSafetyShutoffSubcommand(client *srpc.Client, args []string) {
	if err := clearSafetyShutoff(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error clearing safety shutoff: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func clearSafetyShutoff(client *srpc.Client, subHostname string) error {
	var request dominator.ClearSafetyShutoffRequest
	var reply dominator.ClearSafetyShutoffResponse
	request.Hostname = subHostname
	return client.RequestReply("Dominator.ClearSafetyShutoff", request, &reply)
}
