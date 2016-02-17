package main

import (
	"errors"
	"fmt"
	imclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/objectclient"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"net/rpc"
	"os"
)

func addImageimageSubcommand(args []string) {
	imageClient, imageSClient, objectClient := getClients()
	err := addImageimage(imageClient, imageSClient, objectClient, args[0],
		args[1], args[2], args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addImageimage(imageClient *rpc.Client, imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, oldImageName, filterFilename, triggersFilename string) error {
	var request imageserver.AddImageRequest
	var reply imageserver.AddImageResponse
	imageExists, err := checkImage(imageClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	var newImage image.Image
	if err := loadImageFiles(&newImage, objectClient, filterFilename,
		triggersFilename); err != nil {
		return err
	}
	fs, err := getFsOfImage(imageSClient, oldImageName)
	if err != nil {
		return err
	}
	if fs, err = applyDeleteFilter(fs); err != nil {
		return err
	}
	fs = fs.Filter(newImage.Filter)
	newImage.FileSystem = fs
	request.ImageName = name
	request.Image = &newImage
	err = imclient.CallAddImage(imageSClient, request, &reply)
	if err != nil {
		return errors.New("remote error: " + err.Error())
	}
	return nil
}
