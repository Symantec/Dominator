package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/objectclient"
	"net/rpc"
	"os"
)

func showImageSubcommand(imageClient *rpc.Client,
	objectClient *objectclient.ObjectClient, args []string) {
	err := showImage(imageClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error showing image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func showImage(client *rpc.Client, image string) error {
	fs, err := getImage(client, image)
	if err != nil {
		return err
	}
	return fs.List(os.Stdout)
}
