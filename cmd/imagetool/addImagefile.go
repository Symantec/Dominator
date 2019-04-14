package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/image"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
)

func addImagefileSubcommand(args []string) {
	imageSClient, objectClient := getClients()
	err := addImagefile(imageSClient, objectClient, args[0], args[1], args[2],
		args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\": %s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
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
