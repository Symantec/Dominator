package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/lib/verstr"
)

func listImagesSubcommand(args []string, logger log.DebugLogger) error {
	imageClient, _ := getClients()
	if err := listImages(imageClient); err != nil {
		return fmt.Errorf("Error listing images: %s", err)
	}
	return nil
}

func listImages(imageSClient *srpc.Client) error {
	imageNames, err := client.ListImages(imageSClient)
	if err != nil {
		return err
	}
	verstr.Sort(imageNames)
	for _, name := range imageNames {
		fmt.Println(name)
	}
	return nil
}
