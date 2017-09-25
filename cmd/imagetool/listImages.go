package main

import (
	"fmt"
	"os"
	"path"

	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/verstr"
)

func listImagesSubcommand(args []string) {
	imageClient, _ := getClients()
	if err := listImages(imageClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing images\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listImages(imageSClient *srpc.Client) error {
	imageNames, err := client.ListImages(imageSClient)
	if err != nil {
		return err
	}
	verstr.Sort(imageNames)
	for _, name := range imageNames {
		fmt.Println(name)
	}
	return nil
}

func listLatestImageSubcommand(args []string) {
	imageClient, _ := getClients()
	if err := listLatestImage(imageClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing latest image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listLatestImage(imageSClient *srpc.Client, directory string) error {
	imageNames, err := client.ListImages(imageSClient)
	if err != nil {
		return err
	}
	namesInDirectory := make([]string, 0)
	pattern := path.Join(directory, "*")
	for _, name := range imageNames {
		if matched, _ := path.Match(pattern, name); matched {
			namesInDirectory = append(namesInDirectory, name)
		}
	}
	if len(namesInDirectory) < 1 {
		return nil
	}
	verstr.Sort(namesInDirectory)
	fmt.Println(namesInDirectory[len(namesInDirectory)-1])
	return nil
}
