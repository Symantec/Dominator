package main

import (
	"errors"
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func addImagefileSubcommand(args []string, logger log.DebugLogger) error {
	imageSClient, objectClient := getClients()
	err := addImagefile(imageSClient, objectClient, args[0], args[1], args[2],
		args[3])
	if err != nil {
		return fmt.Errorf("Error adding image: \"%s\": %s", args[0], err)
	}
	return nil
}

func addImagefile(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, imageFilename, filterFilename, triggersFilename string) error {
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
	newImage.FileSystem, err = buildImage(imageSClient, newImage.Filter,
		imageFilename)
	if err != nil {
		return errors.New("error building image: " + err.Error())
	}
	if err := spliceComputedFiles(newImage.FileSystem); err != nil {
		return err
	}
	if err := copyMtimes(imageSClient, newImage, *copyMtimesFrom); err != nil {
		return err
	}
	return addImage(imageSClient, name, newImage)
}

func copyMtimes(imageSClient *srpc.Client, img *image.Image,
	oldImageName string) error {
	if oldImageName == "" {
		return nil
	}
	fs := img.FileSystem
	oldFs, err := getFsOfImage(imageSClient, oldImageName)
	if err != nil {
		return err
	}
	util.CopyMtimes(oldFs, fs)
	return nil
}
