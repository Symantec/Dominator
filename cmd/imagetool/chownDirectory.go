package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func chownDirectorySubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	if err := client.ChownDirectory(imageSClient, args[0],
		args[1]); err != nil {
		return fmt.Errorf("Error changing directory ownership: %s", err)
	}
	return nil
}
