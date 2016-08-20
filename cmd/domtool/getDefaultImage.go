package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/dominator"
	"os"
)

func getDefaultImageSubcommand(client *srpc.Client, args []string) {
	if err := getDefaultImage(client); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting default image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
