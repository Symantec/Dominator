package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
	"path"
)

func unpackImageSubcommand(client *srpc.Client, args []string) {
	if err := unpackImage(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error unpacking image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func unpackImage(client *srpc.Client, imageName string) error {
	imageName = path.Clean(imageName)
	request := proto.UnpackImageRequest{
		path.Dir(imageName), path.Base(imageName)}
	var reply proto.UnpackImageResponse
	return client.RequestReply("ImageUnpacker.UnpackImage", request,
		&reply)
}
