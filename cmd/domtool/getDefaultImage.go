package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func getDefaultImageSubcommand(args []string, logger log.DebugLogger) error {
	if err := getDefaultImage(getClient()); err != nil {
		return fmt.Errorf("Error getting default image: %s", err)
	}
	return nil
}

func getDefaultImage(client *srpc.Client) error {
	var request dominator.GetDefaultImageRequest
	var reply dominator.GetDefaultImageResponse
	if err := client.RequestReply("Dominator.GetDefaultImage", request,
		&reply); err != nil {
		return err
	}
	if reply.ImageName != "" {
		fmt.Println(reply.ImageName)
	}
	return nil
}
