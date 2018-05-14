package main

import (
	"os"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
)

func deleteSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := deletePaths(getSubClient(), args); err != nil {
		logger.Fatalf("Error deleting: %s\n", err)
	}
	os.Exit(0)
}

func deletePaths(srpcClient *srpc.Client, pathnames []string) error {
	return client.CallUpdate(srpcClient, sub.UpdateRequest{
		PathsToDelete: pathnames,
		Wait:          true},
		&sub.UpdateResponse{})
}
