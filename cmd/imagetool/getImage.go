package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/log/nulllogger"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
)

func getImageSubcommand(args []string, logger log.DebugLogger) error {
	_, objectClient := getClients()
	if err := getImageAndWrite(objectClient, args[0], args[1]); err != nil {
		return fmt.Errorf("Error getting image: %s\n", err)
	}
	return nil
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
