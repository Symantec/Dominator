package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func addReplaceImageSubcommand(args []string) {
	imageClient, imageSClient, objectClient := getClients()
	err := addReplaceImage(imageClient, imageSClient, objectClient, args[0],
		args[1], args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addReplaceImage(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, baseImageName string, layerImageNames []string) error {
	var request imageserver.AddImageRequest
	var reply imageserver.AddImageResponse
	imageExists, err := checkImage(imageClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	request.ImageName = name
	request.Image, err = getImage(imageSClient, baseImageName)
	if err != nil {
		return err
	}
	for _, layerImageName := range layerImageNames {
		fs, err := buildImage(objectClient, request.Image.Filter,
			layerImageName)
		if err != nil {
			return err
		}
		if err := layerImages(request.Image.FileSystem, fs); err != nil {
			return err
		}
	}
	err = client.CallAddImage(imageSClient, request, &reply)
	if err != nil {
		return errors.New("remote error: " + err.Error())
	}
	return nil
}

func layerImages(baseFS *filesystem.FileSystem,
	layerFS *filesystem.FileSystem) error {
	for filename, layerInum := range layerFS.FilenameToInodeTable() {
		layerInode := layerFS.InodeTable[layerInum]
		if _, ok := layerInode.(*filesystem.DirectoryInode); ok {
			continue
		}
		baseInum, ok := baseFS.FilenameToInodeTable()[filename]
		if !ok {
			return errors.New(filename + " missing in base image")
		}
		baseInode := baseFS.InodeTable[baseInum]
		sameType, sameMetadata, sameData := filesystem.CompareInodes(baseInode,
			layerInode, nil)
		if !sameType {
			return errors.New(filename + " changed type")
		}
		if sameMetadata && sameData {
			continue
		}
		baseFS.InodeTable[baseInum] = layerInode
	}
	return nil
}
