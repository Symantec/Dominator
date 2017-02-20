package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func cleanupSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := cleanup(getSubClient()); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning up: %s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func cleanup(srpcClient *srpc.Client) error {
	var request sub.PollRequest
	var reply sub.PollResponse
	if err := client.CallPoll(srpcClient, request, &reply); err != nil {
		return err
	}
	if len(reply.ObjectCache) < 1 {
		return nil
	}
	fmt.Fprintf(os.Stderr, "Deleting: %d objects\n", len(reply.ObjectCache))
	return client.Cleanup(srpcClient, reply.ObjectCache)
	return nil
}
