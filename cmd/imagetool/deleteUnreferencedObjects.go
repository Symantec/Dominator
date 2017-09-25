package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Symantec/Dominator/imageserver/client"
)

func deleteUnreferencedObjectsSubcommand(args []string) {
	imageSClient, _ := getClients()
	percentage, err := strconv.ParseUint(args[0], 10, 8)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing percentage: %s\n", err)
		os.Exit(1)
	}
	bytes, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing bytes: %s\n", err)
		os.Exit(1)
	}
	if err := client.DeleteUnreferencedObjects(imageSClient, uint8(percentage),
		bytes); err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting unreferenced objects: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
