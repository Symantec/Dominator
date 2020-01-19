package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func copyImageSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	if err := copyImage(imageSClient, args[0], args[1]); err != nil {
		return fmt.Errorf("Error copying image: %s", err)
	}
	return nil
}

func copyImage(imageSClient *srpc.Client, name, oldImageName string) error {
	imageExists, err := client.CheckImage(imageSClient, name)
	if err != nil {
		return errors.New("error checking for image existence: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	image, err := getImage(imageSClient, oldImageName)
	if err != nil {
		return err
	}
	return addImage(imageSClient, name, image)
}
