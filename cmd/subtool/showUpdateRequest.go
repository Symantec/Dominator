package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/dom/lib"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func showUpdateRequestSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := showUpdateRequest(getSubClient, args[0]); err != nil {
		logger.Fatalf("Error showing update: %s: %s\n", args[0], err)
	}
	os.Exit(0)
}

func showUpdateRequest(getSubClient getSubClientFunc, imageName string) error {
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
	deleteMissingComputedFiles := true
	ignoreMissingComputedFiles := false
	pushComputedFiles := true
	if *computedFilesRoot == "" {
		subObj.ObjectGetter = nullObjectGetterType{}
		deleteMissingComputedFiles = false
		ignoreMissingComputedFiles = true
		pushComputedFiles = false
	} else {
		fs, err := scanner.ScanFileSystem(*computedFilesRoot, nil, nil, nil,
			nil, nil)
		if err != nil {
			return err
		}
		subObj.ObjectGetter = fs
		computedInodes := make(map[string]*filesystem.RegularInode)
		subObj.ComputedInodes = computedInodes
		for filename, inum := range fs.FilenameToInodeTable() {
			if inode, ok := fs.InodeTable[inum].(*filesystem.RegularInode); ok {
				computedInodes[filename] = inode
			}
		}
	}
	objectsToFetch, _ := lib.BuildMissingLists(subObj, img, pushComputedFiles,
		ignoreMissingComputedFiles, logger)
	subObj.ObjectCache = objectcache.ObjectMapToCache(objectsToFetch)
	var updateRequest sub.UpdateRequest
	if lib.BuildUpdateRequest(subObj, img, &updateRequest,
		deleteMissingComputedFiles, ignoreMissingComputedFiles, logger) {
		return errors.New("missing computed file(s)")
	}
	if err := json.WriteWithIndent(os.Stdout, "  ", updateRequest); err != nil {
		return err
	}
	return nil
}
