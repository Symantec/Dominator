package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/dom/lib"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func listMissingObjectsSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClientRetry(logger)
	defer srpcClient.Close()
	if err := listMissingObjects(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error listing missing objects: %s: %s", args[0], err)
	}
	return nil
}

func listMissingObjects(srpcClient *srpc.Client, imageName string) error {
	// Start querying the imageserver for the image.
	imageServerAddress := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	imgChannel := getImageChannel(imageServerAddress, imageName, timeoutTime)
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
