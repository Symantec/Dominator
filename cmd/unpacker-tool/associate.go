package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
)

func associateSubcommand(srpcClient *srpc.Client, args []string) {
	err := client.AssociateStreamWithDevice(srpcClient, args[0], args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error associating: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
