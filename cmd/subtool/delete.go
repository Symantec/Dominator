package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func deleteSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := deletePaths(srpcClient, args); err != nil {
		return fmt.Errorf("Error deleting: %s", err)
	}
	return nil
}

func deletePaths(srpcClient *srpc.Client, pathnames []string) error {
	return client.CallUpdate(srpcClient, sub.UpdateRequest{
		PathsToDelete: pathnames,
		Wait:          true},
		&sub.UpdateResponse{})
}
