package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func setDefaultImageSubcommand(args []string, logger log.DebugLogger) error {
	if err := setDefaultImage(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error setting default image: %s", err)
	}
	return nil
}

func setDefaultImage(client *srpc.Client, imageName string) error {
	var request dominator.SetDefaultImageRequest
	var reply dominator.SetDefaultImageResponse
	request.ImageName = imageName
	if err := client.RequestReply("Dominator.SetDefaultImage", request,
		&reply); err != nil {
		return err
	}
	return nil
}
