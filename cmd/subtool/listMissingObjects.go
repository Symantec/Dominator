package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/dom/lib"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"log"
	"os"
)

func listMissingObjectsSubcommand(getSubClient getSubClientFunc,
	args []string) {
	if err := listMissingObjects(getSubClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error listing missing objects: %s: %s\n",
			args[0], err)
		os.Exit(2)
	}
	os.Exit(0)
}

func listMissingObjects(getSubClient getSubClientFunc, imageName string) error {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	// Start querying the imageserver for the image.
	imageServerAddress := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	imgChannel := getImageChannel(imageServerAddress, imageName, timeoutTime)
	srpcClient := getSubClient()
	subObj := lib.Sub{
		Hostname: *subHostname,
		Client:   srpcClient,
	}
	pollRequest := sub.PollRequest{}
	var pollReply sub.PollResponse
	if err := client.CallPoll(srpcClient, pollRequest, &pollReply); err != nil {
		return err
	}
	fs := pollReply.FileSystem
	if fs == nil {
		return errors.New("sub not ready")
	}
	subObj.FileSystem = fs
	imageResult := <-imgChannel
	img := imageResult.image
	if *filterFile != "" {
		var err error
		img.Filter, err = filter.Load(*filterFile)
		if err != nil {
			return err
		}
	}
	objectsToFetch, _ := lib.BuildMissingLists(subObj, img, false, true,
		logger)
	for hashVal := range objectsToFetch {
		fmt.Printf("%x\n", hashVal)
	}
	return nil
}
