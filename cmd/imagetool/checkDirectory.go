package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
)

func checkDirectorySubcommand(args []string) {
	imageSClient, _ := getClients()
	directoryExists, err := client.CheckDirectory(imageSClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking directory: %s\n", err)
		os.Exit(1)
	}
	if directoryExists {
		os.Exit(0)
	}
	os.Exit(1)
}
