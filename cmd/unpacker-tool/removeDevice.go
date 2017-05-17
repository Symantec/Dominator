package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/srpc"
	"os"
)

func removeDeviceSubcommand(srpcClient *srpc.Client, args []string) {
	if err := client.RemoveDevice(srpcClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing device: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
