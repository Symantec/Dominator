package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"os"
)

func deleteSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := deletePaths(getSubClient(), args); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting: %s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func deletePaths(srpcClient *srpc.Client, pathnames []string) error {
	return srpcClient.RequestReply("Subd.Update", sub.UpdateRequest{
		PathsToDelete: pathnames,
		Wait:          true},
		&sub.FetchResponse{})
}
