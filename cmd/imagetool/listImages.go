package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func listImagesSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	if err := listImages(imageClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listImages(client *rpc.Client) error {
	var request imageserver.ListImagesRequest
	var reply imageserver.ListImagesResponse
	err := client.Call("ImageServer.ListImages", request, &reply)
	if err != nil {
		return err
	}
	for _, name := range reply.ImageNames {
		fmt.Println(name)
	}
	return nil
}
