package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"os"
)

func checkImageSubcommand(args []string) {
	_, imageClient, _ := getClients()
	imageExists, err := client.CallCheckImage(imageClient, args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking image\t%s\n", err)
		os.Exit(1)
	}
	if imageExists {
		os.Exit(0)
	}
	os.Exit(1)
}
