package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func prepareForCopySubcommand(srpcClient *srpc.Client, args []string) error {
	if err := client.PrepareForCopy(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error preparing for copy: %s", err)
	}
	return nil
}
