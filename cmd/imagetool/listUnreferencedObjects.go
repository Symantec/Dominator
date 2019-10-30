package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func listUnreferencedObjectsSubcommand(args []string) {
	imageClient, _ := getClients()
	if err := listUnreferencedObjects(imageClient, false); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing unreferenced objects: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func showUnreferencedObjectsSubcommand(args []string) {
	imageClient, _ := getClients()
	if err := listUnreferencedObjects(imageClient, true); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing unreferenced objects: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listUnreferencedObjects(imageSClient *srpc.Client, showSize bool) error {
	objects, err := client.ListUnreferencedObjects(imageSClient)
	if err != nil {
		return err
	}
	for hashVal, size := range objects {
		if showSize {
			fmt.Printf("%x %d\n", hashVal, size)
		} else {
			fmt.Printf("%x\n", hashVal)
		}
	}
	return nil
}
