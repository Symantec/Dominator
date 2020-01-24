package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func checkDirectorySubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, _ := getClients()
	directoryExists, err := client.CheckDirectory(imageSClient, args[0])
	if err != nil {
		return fmt.Errorf("Error checking directory: %s", err)
	}
	if directoryExists {
		return nil
	}
	os.Exit(1)
	panic("impossible")
}
