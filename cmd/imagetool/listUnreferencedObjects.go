package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func listUnreferencedObjectsSubcommand(args []string,
	logger log.DebugLogger) error {
	imageClient, _ := getClients()
	if err := listUnreferencedObjects(imageClient, false); err != nil {
		return fmt.Errorf("Error listing unreferenced objects: %s", err)
	}
	return nil
}

func showUnreferencedObjectsSubcommand(args []string,
	logger log.DebugLogger) error {
	imageClient, _ := getClients()
	if err := listUnreferencedObjects(imageClient, true); err != nil {
		return fmt.Errorf("Error listing unreferenced objects: %s", err)
	}
	return nil
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
