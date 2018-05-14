package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
)

func restartServiceSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := restartService(getSubClient(), args[0]); err != nil {
		logger.Fatalf("Error deleting: %s\n", err)
	}
	os.Exit(0)
}

func restartService(srpcClient *srpc.Client, serviceName string) error {
	tmpPathname := fmt.Sprintf("/subtool-restart-%d", os.Getpid())
	return client.CallUpdate(srpcClient, sub.UpdateRequest{
		Wait: true,
		InodesToMake: []sub.Inode{
			{
				Name:         tmpPathname,
				GenericInode: &filesystem.RegularInode{},
			},
		},
		PathsToDelete: []string{tmpPathname},
		Triggers: &triggers.Triggers{
			Triggers: []*triggers.Trigger{
				{
					MatchLines: []string{tmpPathname},
					Service:    serviceName,
				},
			},
		},
	},
		&sub.UpdateResponse{})
}
