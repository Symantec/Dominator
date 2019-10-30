package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/log/nulllogger"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
)

func getImageSubcommand(args []string) {
	_, objectClient := getClients()
	err := getImageAndWrite(objectClient, args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getImageAndWrite(objectClient *objectclient.ObjectClient, name,
	dirname string) error {
	fs, objectsGetter, err := getImageForUnpack(objectClient, name)
	if err != nil {
		return err
	}
	return util.Unpack(fs, objectsGetter, dirname, nulllogger.New())
}

func getImageForUnpack(objectClient *objectclient.ObjectClient, name string) (
	*filesystem.FileSystem, objectserver.ObjectsGetter, error) {
	fs, err := getTypedImage(name)
	if err != nil {
		return nil, nil, err
	}
	if *computedFilesRoot == "" {
		return fs, objectClient, nil
	}
	objectsGetter, err := util.ReplaceComputedFiles(fs,
		&util.ComputedFilesData{RootDirectory: *computedFilesRoot},
		objectClient)
	if err != nil {
		return nil, nil, err
	}
	return fs, objectsGetter, nil
}
