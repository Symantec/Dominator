package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func checkImageSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	imageExists, err := client.CheckImage(imageSClient, args[0])
	if err != nil {
		return fmt.Errorf("Error checking image: %s", err)
	}
	if imageExists {
		return nil
	}
	os.Exit(1)
	panic("impossible")
}
