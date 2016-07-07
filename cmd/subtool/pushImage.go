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
	"time"
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
	timeoutTime := time.Now().Add(*timeout)
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
	img, err := getImageRetry(imageServerAddress, imageName, timeoutTime)
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
	subObj := lib.Sub{
		Hostname:       *subHostname,
		Client:         srpcClient,
		ComputedInodes: computedInodes}
	if err := pollFetchAndPush(&subObj, computedObjectGetter, img,
		imageServerAddress, timeoutTime, logger); err != nil {
		return err
	}
	var updateRequest sub.UpdateRequest
	var updateReply sub.UpdateResponse
	if lib.BuildUpdateRequest(subObj, img, &updateRequest, true, logger) {
		return errors.New("missing computed file(s)")
	}
	updateRequest.ImageName = imageName
	updateRequest.Wait = true
	return client.CallUpdate(srpcClient, updateRequest, &updateReply)
}

func getImageRetry(imageServerAddress, imageName string,
	timeoutTime time.Time) (*image.Image, error) {
	for ; time.Now().Before(timeoutTime); time.Sleep(time.Second) {
		img, err := getImage(imageServerAddress, imageName)
		if img != nil && err == nil {
			return img, nil
		}
	}
	return nil, errors.New("timed out getting image")
}

func getImage(imageServerAddress, imageName string) (*image.Image, error) {
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

func pollFetchAndPush(subObj *lib.Sub,
	computedObjectGetter objectserver.ObjectGetter, img *image.Image,
	imageServerAddress string, timeoutTime time.Time,
	logger *log.Logger) error {
	var generationCount uint64
	for ; time.Now().Before(timeoutTime); time.Sleep(time.Second) {
		var pollReply sub.PollResponse
		if err := pollAndBuildPointers(subObj.Client, &generationCount,
			&pollReply); err != nil {
			return err
		}
		if pollReply.FileSystem == nil {
			continue
		}
		subObj.FileSystem = pollReply.FileSystem
		subObj.ObjectCache = pollReply.ObjectCache
		objectsToFetch, objectsToPush := lib.BuildMissingLists(*subObj, img,
			true, true, logger)
		if len(objectsToFetch) < 1 && len(objectsToPush) < 1 {
			return nil
		}
		if len(objectsToFetch) > 0 {
			err := subObj.Client.RequestReply("Subd.Fetch", sub.FetchRequest{
				ServerAddress: imageServerAddress,
				Wait:          true,
				Hashes:        objectsToFetch},
				&sub.FetchResponse{})
			if err != nil {
				logger.Printf("Error calling %s:Subd.Fetch(): %s\n",
					subHostname, err)
				return err
			}
		}
		if len(objectsToPush) > 0 {
			err := lib.PushObjects(*subObj, objectsToPush, computedObjectGetter,
				logger)
			if err != nil {
				return err
			}
		}
	}
	return errors.New("timed out fetching and pushing objects")
}

func pollAndBuildPointers(srpcClient *srpc.Client, generationCount *uint64,
	pollReply *sub.PollResponse) error {
	pollRequest := sub.PollRequest{HaveGeneration: *generationCount}
	err := client.CallPoll(srpcClient, pollRequest, pollReply)
	if err != nil {
		return err
	}
	*generationCount = pollReply.GenerationCount
	fs := pollReply.FileSystem
	if fs == nil {
		return nil
	}
	if err := fs.RebuildInodePointers(); err != nil {
		return err
	}
	fs.BuildEntryMap()
	return nil
}
