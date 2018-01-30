package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/filesystem/util"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
)

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
	return util.WriteRaw(fs, objectsGetter, rawFilename, tableType,
		*minFreeBytes, *roundupPower, *makeBootable, logger)
}
