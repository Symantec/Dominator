package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"os"
)

func deleteImageSubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := client.CallDeleteImage(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
