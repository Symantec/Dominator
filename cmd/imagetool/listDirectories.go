package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func listDirectoriesSubcommand(args []string, logger log.DebugLogger) error {
	imageClient, _ := getClients()
	if err := listDirectories(imageClient); err != nil {
		return fmt.Errorf("Error listing directories: %s", err)
	}
	return nil
}

func listDirectories(imageSClient *srpc.Client) error {
	directories, err := client.ListDirectories(imageSClient)
	if err != nil {
		return err
	}
	image.SortDirectories(directories)
	maxDirnameWidth := 0
	for _, directory := range directories {
		if len(directory.Name) > maxDirnameWidth {
			maxDirnameWidth = len(directory.Name)
		}
	}
	for _, directory := range directories {
		if directory.Metadata.OwnerGroup == "" {
			fmt.Println(directory.Name)
			continue
		}
		fmt.Printf("%-*s  ", maxDirnameWidth, directory.Name)
		fmt.Printf("OwnerGroup=%s", directory.Metadata.OwnerGroup)
		fmt.Println()
	}
	return nil
}
