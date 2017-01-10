package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
)

func prepareForUnpackSubcommand(srpcClient *srpc.Client, args []string) {
	if err := prepareForUnpack(srpcClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing for unpack: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func prepareForUnpack(srpcClient *srpc.Client, streamName string) error {
	return client.PrepareForUnpack(srpcClient, streamName, false, false)
}
