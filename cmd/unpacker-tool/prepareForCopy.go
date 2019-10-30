package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func prepareForCopySubcommand(srpcClient *srpc.Client, args []string) {
	if err := client.PrepareForCopy(srpcClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing for copy: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
