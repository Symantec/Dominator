package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/dom/lib"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"log"
	"os"
)

func showUpdateRequestSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := showUpdateRequest(getSubClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error showing update: %s: %s\n", args[0], err)
		os.Exit(2)
	}
	os.Exit(0)
}

func showUpdateRequest(getSubClient getSubClientFunc, imageName string) error {
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
	if err := fs.RebuildInodePointers(); err != nil {
		return err
	}
	fs.BuildEntryMap()
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
	subObj.ObjectCache, _ = lib.BuildMissingLists(subObj, img, false, true,
		logger)
	var updateRequest sub.UpdateRequest
	if lib.BuildUpdateRequest(subObj, img, &updateRequest, true, logger) {
		return errors.New("missing computed file(s)")
	}
	if err := json.WriteWithIndent(os.Stdout, "  ", updateRequest); err != nil {
		return err
	}
	return nil
}
