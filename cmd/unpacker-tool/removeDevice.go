package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func removeDeviceSubcommand(srpcClient *srpc.Client, args []string) error {
	if err := client.RemoveDevice(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error removing device: %s", err)
	}
	return nil
}
