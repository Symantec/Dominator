package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageserver/client"
)

func deleteImageSubcommand(args []string) {
	imageSClient, _ := getClients()
	if err := client.DeleteImage(imageSClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting image\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
