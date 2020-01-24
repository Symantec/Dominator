package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func deleteImageSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	if err := client.DeleteImage(imageSClient, args[0]); err != nil {
		return fmt.Errorf("Error deleting image: %s", err)
	}
	return nil
}
