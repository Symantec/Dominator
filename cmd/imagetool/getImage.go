package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/log/nulllogger"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
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
	fs, err := getTypedImage(name)
	if err != nil {
		return err
	}
	return util.Unpack(fs, objectClient, dirname, nulllogger.New())
}
