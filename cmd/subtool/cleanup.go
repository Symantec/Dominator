package main

import (
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func cleanupSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := cleanup(getSubClient()); err != nil {
		logger.Fatalf("Error cleaning up: %s\n", err)
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
	logger.Printf("Deleting: %d objects\n", len(reply.ObjectCache))
	return client.Cleanup(srpcClient, reply.ObjectCache)
}
