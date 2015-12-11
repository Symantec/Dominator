package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"net/rpc"
	"os"
)

func showImageSubcommand(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	if err := showImage(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error showing image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func showImage(client *srpc.Client, image string) error {
	fs, err := getImage(client, image)
	if err != nil {
		return err
	}
	return fs.List(os.Stdout)
}
