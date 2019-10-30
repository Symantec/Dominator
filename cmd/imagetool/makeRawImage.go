package main

import (
	"fmt"
	"os"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem/util"
	objectclient "github.com/Cloud-Foundations/Dominator/lib/objectserver/client"
)

const filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP |
	syscall.S_IROTH

func makeRawImageSubcommand(args []string) {
	_, objectClient := getClients()
	err := makeRawImage(objectClient, args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making raw image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
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
