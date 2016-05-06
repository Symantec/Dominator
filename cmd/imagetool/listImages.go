package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/verstr"
	"os"
)

func listImagesSubcommand(args []string) {
	imageClient, _ := getClients()
	if err := listImages(imageClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listImages(imageSClient *srpc.Client) error {
	imageNames, err := client.CallListImages(imageSClient)
	if err != nil {
		return err
	}
	verstr.Sort(imageNames)
	for _, name := range imageNames {
		fmt.Println(name)
	}
	return nil
}
