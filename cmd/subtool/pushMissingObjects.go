package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/dom/lib"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/objectcache"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func pushMissingObjectsSubcommand(getSubClient getSubClientFunc,
	args []string) {
	if err := pushMissingObjects(getSubClient, args[0]); err != nil {
		logger.Fatalf("Error pushing missing objects: %s: %s\n", args[0], err)
	}
	os.Exit(0)
}

func pushMissingObjects(getSubClient getSubClientFunc, imageName string) error {
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
	objSrv := objectclient.NewObjectClient(fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum))
	subObjClient := objectclient.AttachObjectClient(srpcClient)
	defer subObjClient.Close()
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
	hashes := objectcache.ObjectMapToCache(objectsToFetch)
	objectsReader, err := objSrv.GetObjects(hashes)
	if err != nil {
		return err
	}
	defer objectsReader.Close()
	for _, hashVal := range hashes {
		fmt.Printf("%x\n", hashVal)
		length, reader, err := objectsReader.NextObject()
		if err != nil {
			return err
		}
		_, _, err = subObjClient.AddObject(reader, length, nil)
		reader.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
