package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func makeDirectorySubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	if err := client.MakeDirectory(imageSClient, args[0]); err != nil {
		return fmt.Errorf("Error creating directory: %s\n", err)
	}
	return nil
}
