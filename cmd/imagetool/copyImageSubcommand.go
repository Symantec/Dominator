package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
)

func copyImageSubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := copyImage(imageSClient, args[0], args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error copying image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func copyImage(imageSClient *srpc.Client, name, oldImageName string) error {
	imageExists, err := client.CheckImage(imageSClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	image, err := getImage(imageSClient, oldImageName)
	return addImage(imageSClient, name, image)
}
