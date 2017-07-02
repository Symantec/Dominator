package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem/util"
	"github.com/Symantec/Dominator/lib/log/nulllogger"
	objectclient "github.com/Symantec/Dominator/lib/objectserver/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
)

func getImageSubcommand(args []string) {
	imageSClient, objectClient := getClients()
	err := getImageAndWrite(imageSClient, objectClient, args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getImageAndWrite(imageClient *srpc.Client,
	objectClient *objectclient.ObjectClient, name, dirname string) error {
	fs, err := getFsOfImage(imageClient, name)
	if err != nil {
		return err
	}
	return util.Unpack(fs, objectClient, dirname, nulllogger.New())
}
