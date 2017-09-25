package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/srpc"
)

func unpackImageSubcommand(srpcClient *srpc.Client, args []string) {
	if err := client.UnpackImage(srpcClient, args[0], args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error unpacking image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
