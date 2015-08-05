package main

import (
	"fmt"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func listImagesSubcommand(client *rpc.Client, args []string) {
	err := listImages(client)
	if err != nil {
		fmt.Printf("Error listing images\t%s\n", err)
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
