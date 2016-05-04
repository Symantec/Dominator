package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/filter"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	subclient "github.com/Symantec/Dominator/sub/client"
	"io"
	"net/rpc"
	"os"
)

func addImagesubSubcommand(args []string) {
	imageClient, imageSClient, objectClient := getClients()
	err := addImagesub(imageClient, imageSClient, objectClient, args[0],
		args[1], args[2], args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImagesub(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, subName, filterFilename, triggersFilename string) error {
	imageExists, err := client.CallCheckImage(imageSClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	newImage := new(image.Image)
	if err := loadImageFiles(newImage, objectClient, filterFilename,
		triggersFilename); err != nil {
		return err
	}
	fs, err := pollImage(subName)
	if err != nil {
		return err
	}
	if fs, err = applyDeleteFilter(fs); err != nil {
		return err
	}
	fs = fs.Filter(newImage.Filter)
	if err := spliceComputedFiles(fs); err != nil {
		return err
	}
	if err := copyMissingObjects(fs, imageSClient, objectClient,
		subName); err != nil {
		return err
	}
	newImage.FileSystem = fs
	return addImage(imageSClient, name, newImage)
}

func applyDeleteFilter(fs *filesystem.FileSystem) (
	*filesystem.FileSystem, error) {
	if *deleteFilter == "" {
		return fs, nil
	}
	filter, err := filter.LoadFilter(*deleteFilter)
	if err != nil {
		return nil, err
	}
	return fs.Filter(filter), nil
}

func copyMissingObjects(fs *filesystem.FileSystem, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, subName string) error {
	// Check to see which objects are in the objectserver.
	hashes := make([]hash.Hash, 0, fs.NumRegularInodes)
	for hash, _ := range fs.HashToInodesTable() {
		hashes = append(hashes, hash)
	}
	objectSizes, err := objectClient.CheckObjects(hashes)
	if err != nil {
		return err
	}
	missingHashes := make([]hash.Hash, 0)
	for index, size := range objectSizes {
		if size < 1 {
			missingHashes = append(missingHashes, hashes[index])
		}
	}
	if len(missingHashes) < 1 {
		return nil
	}
	// Get missing objects from sub.
	filesForMissingObjects := make([]string, 0, len(missingHashes))
	for _, hash := range missingHashes {
		if inums, ok := fs.HashToInodesTable()[hash]; !ok {
			return fmt.Errorf("no inode for object: %x", hash)
		} else if files, ok := fs.InodeToFilenamesTable()[inums[0]]; !ok {
			return fmt.Errorf("no file for inode: %d", inums[0])
		} else {
			filesForMissingObjects = append(filesForMissingObjects, files[0])
		}
	}
	objAdderQueue, err := objectclient.NewObjectAdderQueue(imageSClient)
	if err != nil {
		return err
	}
	subClient, err := srpc.DialHTTP("tcp",
		fmt.Sprintf("%s:%d", subName, constants.SubPortNumber), 0)
	if err != nil {
		return fmt.Errorf("error dialing %s", err)
	}
	defer subClient.Close()
	if err := subclient.GetFiles(subClient, filesForMissingObjects,
		func(reader io.Reader, size uint64) error {
			_, err := objAdderQueue.Add(reader, size)
			return err
		}); err != nil {
		return err
	}
	return objAdderQueue.Close()
}
