package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func clearSafetyShutoffSubcommand(args []string, logger log.DebugLogger) error {
	if err := clearSafetyShutoff(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error clearing safety shutoff: %s", err)
	}
	return nil
}

func clearSafetyShutoff(client *srpc.Client, subHostname string) error {
	var request dominator.ClearSafetyShutoffRequest
	var reply dominator.ClearSafetyShutoffResponse
	request.Hostname = subHostname
	return client.RequestReply("Dominator.ClearSafetyShutoff", request, &reply)
}
