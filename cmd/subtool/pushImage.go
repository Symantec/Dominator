package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/dom/lib"
	imgclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"io"
	"log"
	"os"
)

type nullObjectGetterType struct{}

func (getter nullObjectGetterType) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return 0, nil, errors.New("no computed files")
}

func pushImageSubcommand(srpcClient *srpc.Client, args []string) {
	if err := pushImage(srpcClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error pushing image: %s: %s\n", args[0], err)
		os.Exit(2)
	}
	os.Exit(0)
}

func pushImage(srpcClient *srpc.Client, imageName string) error {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	computedInodes := make(map[string]*filesystem.RegularInode)
	var computedObjectGetter objectserver.ObjectGetter
	if *computedFilesRoot == "" {
		computedObjectGetter = nullObjectGetterType{}
	} else {
		fs, err := scanner.ScanFileSystem(*computedFilesRoot, nil, nil, nil,
			nil, nil)
		if err != nil {
			return err
		}
		computedObjectGetter = fs
		for filename, inum := range fs.FilenameToInodeTable() {
			if inode, ok := fs.InodeTable[inum].(*filesystem.RegularInode); ok {
				computedInodes[filename] = inode
			}
		}
	}
	imageServerAddress := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	img, err := getImage(imageServerAddress, imageName)
	if err != nil {
		return err
	}
	if *filterFile != "" {
		img.Filter, err = filter.Load(*filterFile)
		if err != nil {
			return err
		}
	}
	if *triggersFile != "" {
		img.Triggers, err = triggers.Load(*triggersFile)
		if err != nil {
			return err
		}
	}
	// TODO(rgooch): Put poll&fetch in a retry loop: iterate before update.
	var pollReply sub.PollResponse
	if err := pollAndBuildPointers(srpcClient, &pollReply); err != nil {
		return err
	}
	subObj := lib.Sub{
		Hostname:       *subHostname,
		Client:         srpcClient,
		FileSystem:     pollReply.FileSystem,
		ComputedInodes: computedInodes,
		ObjectCache:    pollReply.ObjectCache}
	objectsToFetch, objectsToPush := lib.BuildMissingLists(subObj, img, true,
		true, logger)
	if len(objectsToFetch) > 0 {
		err := srpcClient.RequestReply("Subd.Fetch", sub.FetchRequest{
			ServerAddress: imageServerAddress,
			Wait:          true,
			Hashes:        objectsToFetch},
			&sub.FetchResponse{})
		if err != nil {
			logger.Printf("Error calling %s:Subd.Fetch(): %s\n", subHostname,
				err)
			return err
		}
	}
	if len(objectsToPush) > 0 {
		err := lib.PushObjects(subObj, objectsToPush, computedObjectGetter,
			logger)
		if err != nil {
			return nil
		}
	}
	if err := pollAndBuildPointers(srpcClient, &pollReply); err != nil {
		return err
	}
	subObj.ObjectCache = pollReply.ObjectCache
	var updateRequest sub.UpdateRequest
	var updateReply sub.UpdateResponse
	if lib.BuildUpdateRequest(subObj, img, &updateRequest, true, logger) {
		return errors.New("missing computed file(s)")
	}
	updateRequest.ImageName = imageName
	updateRequest.Wait = true
	return client.CallUpdate(srpcClient, updateRequest, &updateReply)
}

func getImage(imageServerAddress, imageName string) (*image.Image, error) {
	// TODO(rgooch): Put everything below in a retry loop.
	imageSrpcClient, err := srpc.DialHTTP("tcp", imageServerAddress, 0)
	if err != nil {
		return nil, err
	}
	defer imageSrpcClient.Close()
	img, err := imgclient.GetImage(imageSrpcClient, imageName)
	if err != nil {
		return nil, err
	}
	if img == nil {
		return nil, errors.New(imageName + ": not found")
	}
	if err := img.FileSystem.RebuildInodePointers(); err != nil {
		return nil, err
	}
	img.FileSystem.InodeToFilenamesTable()
	img.FileSystem.FilenameToInodeTable()
	img.FileSystem.HashToInodesTable()
	img.FileSystem.ComputeTotalDataBytes()
	img.FileSystem.BuildEntryMap()
	return img, nil
}

func pollAndBuildPointers(srpcClient *srpc.Client,
	pollReply *sub.PollResponse) error {
	pollRequest := sub.PollRequest{}
	err := client.CallPoll(srpcClient, pollRequest, pollReply)
	if err != nil {
		return err
	}
	fs := pollReply.FileSystem
	if fs == nil {
		return errors.New("no file-system data")
	}
	if err := fs.RebuildInodePointers(); err != nil {
		return err
	}
	fs.BuildEntryMap()
	return nil
}
