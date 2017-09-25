package main

import (
	"os"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func deleteSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := deletePaths(getSubClient(), args); err != nil {
		logger.Fatalf("Error deleting: %s\n", err)
	}
	os.Exit(0)
}

func deletePaths(srpcClient *srpc.Client, pathnames []string) error {
	return srpcClient.RequestReply("Subd.Update", sub.UpdateRequest{
		PathsToDelete: pathnames,
		Wait:          true},
		&sub.FetchResponse{})
}
