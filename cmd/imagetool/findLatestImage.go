package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func findLatestImageSubcommand(args []string, logger log.DebugLogger) error {
	if err := findLatestImage(args[0]); err != nil {
		return fmt.Errorf("Error finding latest image: %s\n", err)
	}
	return nil
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
