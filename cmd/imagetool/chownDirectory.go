package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"os"
)

func chownDirectorySubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := client.CallChownDirectory(imageSClient, args[0],
		args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error changing directory ownership: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
