package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func cleanupSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := cleanup(srpcClient); err != nil {
		return fmt.Errorf("Error cleaning up: %s", err)
	}
	return nil
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
