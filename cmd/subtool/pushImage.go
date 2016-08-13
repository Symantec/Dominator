package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/dom/lib"
	imgclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filesystem/scanner"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/format"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
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

type timedImageFetch struct {
	image    *image.Image
	duration time.Duration
}

func (getter nullObjectGetterType) GetObject(hashVal hash.Hash) (
	uint64, io.ReadCloser, error) {
	return 0, nil, errors.New("no computed files")
}

func pushImageSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := pushImage(getSubClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error pushing image: %s: %s\n", args[0], err)
		os.Exit(2)
	}
	os.Exit(0)
}

func pushImage(getSubClient getSubClientFunc, imageName string) error {
	logger := log.New(os.Stderr, "", log.LstdFlags)
	computedInodes := make(map[string]*filesystem.RegularInode)
	// Start querying the imageserver for the image.
	imageServerAddress := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	imgChannel := getImageChannel(imageServerAddress, imageName, timeoutTime)
	startTime := showStart("getSubClient()")
	srpcClient := getSubClient()
	showTimeTaken(startTime)
	subObj := lib.Sub{
		Hostname:       *subHostname,
		Client:         srpcClient,
		ComputedInodes: computedInodes}
	if *computedFilesRoot == "" {
		subObj.ObjectGetter = nullObjectGetterType{}
	} else {
		fs, err := scanner.ScanFileSystem(*computedFilesRoot, nil, nil, nil,
			nil, nil)
		if err != nil {
			return err
		}
		subObj.ObjectGetter = fs
		for filename, inum := range fs.FilenameToInodeTable() {
			if inode, ok := fs.InodeTable[inum].(*filesystem.RegularInode); ok {
				computedInodes[filename] = inode
			}
		}
	}
	startTime = showStart("<-imgChannel")
	imageResult := <-imgChannel
	showTimeTaken(startTime)
	fmt.Fprintf(os.Stderr, "Background image fetch took %s\n",
		format.Duration(imageResult.duration))
	img := imageResult.image
	var err error
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
	} else if *triggersString != "" {
		img.Triggers, err = triggers.Decode([]byte(*triggersString))
		if err != nil {
			return err
		}
	}
	if err := pollFetchAndPush(&subObj, img, imageServerAddress, timeoutTime,
		logger); err != nil {
		return err
	}
	var updateRequest sub.UpdateRequest
	var updateReply sub.UpdateResponse
	startTime = showStart("lib.BuildUpdateRequest()")
	if lib.BuildUpdateRequest(subObj, img, &updateRequest, true, logger) {
		showBlankLine()
		return errors.New("missing computed file(s)")
	}
	showTimeTaken(startTime)
	updateRequest.ImageName = imageName
	updateRequest.Wait = true
	startTime = showStart("Subd.Update()")
	err = client.CallUpdate(srpcClient, updateRequest, &updateReply)
	if err != nil {
		showBlankLine()
		return err
	}
	showTimeTaken(startTime)
	return nil
}

func getImageChannel(clientName, imageName string,
	timeoutTime time.Time) <-chan timedImageFetch {
	resultChannel := make(chan timedImageFetch, 1)
	go func() {
		startTime := time.Now()
		img, err := getImageRetry(clientName, imageName, timeoutTime)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting image: %s\n", err)
			os.Exit(1)
		}
		resultChannel <- timedImageFetch{img, time.Since(startTime)}
	}()
	return resultChannel
}

func getImageRetry(clientName, imageName string,
	timeoutTime time.Time) (*image.Image, error) {
	imageSrpcClient, err := srpc.DialHTTP("tcp", clientName, 0)
	if err != nil {
		return nil, err
	}
	defer imageSrpcClient.Close()
	for ; time.Now().Before(timeoutTime); time.Sleep(time.Second) {
		img, err := imgclient.GetImage(imageSrpcClient, imageName)
		if err != nil {
			return nil, err
		} else if img != nil {
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
	}
	return nil, errors.New("timed out getting image")
}

func pollFetchAndPush(subObj *lib.Sub, img *image.Image,
	imageServerAddress string, timeoutTime time.Time,
	logger *log.Logger) error {
	var generationCount uint64
	deleteEarly := *deleteBeforeFetch
	for ; time.Now().Before(timeoutTime); time.Sleep(time.Second) {
		var pollReply sub.PollResponse
		if err := pollAndBuildPointers(subObj.Client, &generationCount,
			&pollReply); err != nil {
			return err
		}
		if pollReply.FileSystem == nil {
			continue
		}
		if deleteEarly {
			deleteEarly = false
			if deleteUnneededFiles(subObj.Client, pollReply.FileSystem,
				img.FileSystem, logger) {
				continue
			}
		}
		subObj.FileSystem = pollReply.FileSystem
		subObj.ObjectCache = pollReply.ObjectCache
		startTime := showStart("lib.BuildMissingLists()")
		objectsToFetch, objectsToPush := lib.BuildMissingLists(*subObj, img,
			true, true, logger)
		showTimeTaken(startTime)
		if len(objectsToFetch) < 1 && len(objectsToPush) < 1 {
			return nil
		}
		if len(objectsToFetch) > 0 {
			startTime := showStart("Fetch()")
			err := subObj.Client.RequestReply("Subd.Fetch", sub.FetchRequest{
				ServerAddress: imageServerAddress,
				Wait:          true,
				Hashes:        objectsToFetch},
				&sub.FetchResponse{})
			if err != nil {
				showBlankLine()
				logger.Printf("Error calling %s:Subd.Fetch(%s): %s\n",
					subObj.Hostname, imageServerAddress, err)
				return err
			}
			showTimeTaken(startTime)
		}
		if len(objectsToPush) > 0 {
			startTime := showStart("lib.PushObjects()")
			err := lib.PushObjects(*subObj, objectsToPush, logger)
			if err != nil {
				showBlankLine()
				return err
			}
			showTimeTaken(startTime)
		}
	}
	return errors.New("timed out fetching and pushing objects")
}

func pollAndBuildPointers(srpcClient *srpc.Client, generationCount *uint64,
	pollReply *sub.PollResponse) error {
	pollRequest := sub.PollRequest{HaveGeneration: *generationCount}
	startTime := showStart("Poll()")
	err := client.CallPoll(srpcClient, pollRequest, pollReply)
	if err != nil {
		showBlankLine()
		return err
	}
	showTimeTaken(startTime)
	*generationCount = pollReply.GenerationCount
	fs := pollReply.FileSystem
	if fs == nil {
		return nil
	}
	startTime = showStart("FileSystem.RebuildInodePointers()")
	if err := fs.RebuildInodePointers(); err != nil {
		showBlankLine()
		return err
	}
	showTimeTaken(startTime)
	fs.BuildEntryMap()
	return nil
}

func showStart(operation string) time.Time {
	if *showTimes {
		fmt.Fprint(os.Stderr, operation, " ")
	}
	return time.Now()
}

func showTimeTaken(startTime time.Time) {
	if *showTimes {
		stopTime := time.Now()
		fmt.Fprintf(os.Stderr, "took %s\n",
			format.Duration(stopTime.Sub(startTime)))
	}
}

func showBlankLine() {
	if *showTimes {
		fmt.Fprintln(os.Stderr)
	}
}

func deleteUnneededFiles(srpcClient *srpc.Client, subFS *filesystem.FileSystem,
	imgFS *filesystem.FileSystem, logger *log.Logger) bool {
	startTime := showStart("compute early files to delete")
	pathsToDelete := make([]string, 0)
	imgHashToInodesTable := imgFS.HashToInodesTable()
	for pathname, inum := range subFS.FilenameToInodeTable() {
		if inode, ok := subFS.InodeTable[inum].(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				if _, ok := imgHashToInodesTable[inode.Hash]; !ok {
					pathsToDelete = append(pathsToDelete, pathname)
				}
			}
		}
	}
	showTimeTaken(startTime)
	if len(pathsToDelete) < 1 {
		return false
	}
	updateRequest := sub.UpdateRequest{
		Wait:          true,
		PathsToDelete: pathsToDelete}
	var updateReply sub.UpdateResponse
	startTime = showStart("Subd.Update() for early files to delete")
	err := client.CallUpdate(srpcClient, updateRequest, &updateReply)
	showTimeTaken(startTime)
	if err != nil {
		logger.Println(err)
	}
	return true
}
