package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func deleteImageSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := deleteImage(imageClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func deleteImage(client *rpc.Client, name string) error {
	var request imageserver.DeleteImageRequest
	request.ImageName = name
	var reply imageserver.DeleteImageResponse
	err := client.Call("ImageServer.DeleteImage", request, &reply)
	if err != nil {
		return err
	}
	if reply.Success {
		return nil
	}
	return errors.New(reply.ErrorString)
}
