package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageserver/client"
)

func findLatestImageSubcommand(args []string) {
	if err := findLatestImage(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error finding latest image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func findLatestImage(dirname string) error {
	imageSClient, _ := getClients()
	imageName, err := client.FindLatestImage(imageSClient, dirname,
		*ignoreExpiring)
	if err != nil {
		return err
	}
	if imageName == "" {
		return errors.New("no image found")
	}
	fmt.Println(imageName)
	return nil
}
