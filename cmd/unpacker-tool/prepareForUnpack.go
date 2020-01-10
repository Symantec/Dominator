package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func prepareForUnpackSubcommand(srpcClient *srpc.Client, args []string) error {
	if err := prepareForUnpack(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error preparing for unpack: %s", err)
	}
	return nil
}

func prepareForUnpack(srpcClient *srpc.Client, streamName string) error {
	return client.PrepareForUnpack(srpcClient, streamName, false, false)
}
