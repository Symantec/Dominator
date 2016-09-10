package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
	"strconv"
	"strings"
)

func addReplaceImageSubcommand(args []string) {
	imageSClient, objectClient := getClients()
	err := addReplaceImage(imageSClient, objectClient, args[0], args[1],
		args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func bulkAddReplaceImagesSubcommand(args []string) {
	imageSClient, objectClient := getClients()
	err := bulkAddReplaceImages(imageSClient, objectClient, args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error adding image: \"%s\"\t%s\n", args[0], err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addReplaceImage(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient,
	name, baseImageName string, layerImageNames []string) error {
	imageExists, err := client.CheckImage(imageSClient, name)
	if err != nil {
		return errors.New("error checking for image existance: " + err.Error())
	}
	if imageExists {
		return errors.New("image exists")
	}
	newImage, err := getImage(imageSClient, baseImageName)
	if err != nil {
		return err
	}
	if !newImage.ExpiresAt.IsZero() {
		fmt.Fprintf(os.Stderr, "Skipping expiring image: %s\n", baseImageName)
		return nil
	}
	for _, layerImageName := range layerImageNames {
		fs, err := buildImage(imageSClient, newImage.Filter, layerImageName)
		if err != nil {
			return err
		}
		if err := layerImages(newImage.FileSystem, fs); err != nil {
			return err
		}
	}
	if err := spliceComputedFiles(newImage.FileSystem); err != nil {
		return err
	}
	return addImage(imageSClient, name, newImage)
}

func bulkAddReplaceImages(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, layerImageNames []string) error {
	imageNames, err := client.ListImages(imageSClient)
	if err != nil {
		return err
	}
	err = bulkAddReplaceImagesSep(imageSClient, objectClient, layerImageNames,
		imageNames, "/")
	if err != nil {
		return err
	}
	return bulkAddReplaceImagesSep(imageSClient, objectClient, layerImageNames,
		imageNames, ".")
}

func bulkAddReplaceImagesSep(imageSClient *srpc.Client,
	objectClient *objectclient.ObjectClient, layerImageNames []string,
	imageNames []string, separator string) error {
	baseNames := make(map[string]uint64)
	for _, name := range imageNames {
		fields := strings.Split(name, separator)
		nFields := len(fields)
		if nFields < 2 {
			continue
		}
		lastField := fields[nFields-1]
		if version, err := strconv.ParseUint(lastField, 10, 64); err != nil {
			continue
		} else {
			name := strings.Join(fields[:nFields-1], separator)
			if oldVersion := baseNames[name]; version >= oldVersion {
				baseNames[name] = version
			}
		}
	}
	for baseName, version := range baseNames {
		oldName := fmt.Sprintf("%s%s%d", baseName, separator, version)
		newName := fmt.Sprintf("%s%s%d", baseName, separator, version+1)
		if err := addReplaceImage(imageSClient, objectClient,
			newName, oldName, layerImageNames); err != nil {
			return err
		}
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
