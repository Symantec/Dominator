package main

import (
	"fmt"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
)

const filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
	syscall.S_IROTH

func makeRawImageSubcommand(args []string, logger log.DebugLogger) error {
	_, objectClient := getClients()
	if err := makeRawImage(objectClient, args[0], args[1]); err != nil {
		return fmt.Errorf("Error making raw image: %s", err)
	}
	return nil
}

func makeRawImage(objectClient *objectclient.ObjectClient, name,
	rawFilename string) error {
	fs, objectsGetter, err := getImageForUnpack(objectClient, name)
	if err != nil {
		return err
	}
	return util.WriteRaw(fs, objectsGetter, rawFilename, filePerms, tableType,
		*minFreeBytes, *roundupPower, *makeBootable, *allocateBlocks, logger)
}
