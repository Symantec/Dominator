package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageserver/client"
)

func makeDirectorySubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := client.MakeDirectory(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating directory: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
