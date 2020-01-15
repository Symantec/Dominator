package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func addImageimageSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, objectClient := getClients()
	err := addImageimage(imageSClient, objectClient, args[0], args[1], args[2],
		args[3])
	if err != nil {
		return fmt.Errorf("Error adding image: \"%s\"\t%s\n", args[0], err)
	}
	return nil
}

func addImageimage(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, oldImageName, filterFilename, triggersFilename string) error {
	imageExists, err := client.CheckImage(imageSClient, name)
	if err != nil {
		return errors.New("error checking for image existence: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	newImage := new(image.Image)
	if err := loadImageFiles(newImage, objectClient, filterFilename,
		triggersFilename); err != nil {
		return err
	}
	fs, err := getFsOfImage(imageSClient, oldImageName)
	if err != nil {
		return err
	}
	if err := spliceComputedFiles(fs); err != nil {
		return err
	}
	if fs, err = applyDeleteFilter(fs); err != nil {
		return err
	}
	fs = fs.Filter(newImage.Filter)
	newImage.FileSystem = fs
	return addImage(imageSClient, name, newImage)
}
