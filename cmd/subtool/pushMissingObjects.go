package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/dom/lib"
	"github.com/Cloud-Foundations/Dominator/lib/filter"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func pushMissingObjectsSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClientRetry(logger)
	defer srpcClient.Close()
	if err := pushMissingObjects(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error pushing missing objects: %s: %s", args[0], err)
	}
	return nil
}

func pushMissingObjects(srpcClient *srpc.Client, imageName string) error {
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
