package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
)

func chownDirectorySubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := client.ChownDirectory(imageSClient, args[0],
		args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing directory ownership: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
