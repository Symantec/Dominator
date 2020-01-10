package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func associateSubcommand(srpcClient *srpc.Client, args []string) error {
	err := client.AssociateStreamWithDevice(srpcClient, args[0], args[1])
	if err != nil {
		return fmt.Errorf("Error associating: %s", err)
	}
	return nil
}
