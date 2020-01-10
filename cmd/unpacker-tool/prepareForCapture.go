package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func prepareForCaptureSubcommand(srpcClient *srpc.Client, args []string) error {
	if err := client.PrepareForCapture(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error preparing for capture: %s", err)
	}
	return nil
}
