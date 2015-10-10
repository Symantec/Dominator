package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func checkImageSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	imageExists, err := checkImage(imageClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking image\t%s\n", err)
		os.Exit(1)
	}
	if imageExists {
		os.Exit(0)
	}
	os.Exit(1)
}

func checkImage(client *rpc.Client, name string) (bool, error) {
	var request imageserver.CheckImageRequest
	request.ImageName = name
	var reply imageserver.CheckImageResponse
	err := client.Call("ImageServer.CheckImage", request, &reply)
	if err != nil {
		return false, err
	}
	return reply.ImageExists, nil
}
