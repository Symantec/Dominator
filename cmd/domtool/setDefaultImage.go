package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func setDefaultImageSubcommand(client *srpc.Client, args []string) {
	if err := setDefaultImage(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting default image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
