package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func restartServiceSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := restartService(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error deleting: %s", err)
	}
	return nil
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
