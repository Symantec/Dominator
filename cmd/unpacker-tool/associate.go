package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func associateSubcommand(args []string, logger log.DebugLogger) error {
	err := client.AssociateStreamWithDevice(getClient(), args[0], args[1])
	if err != nil {
		return fmt.Errorf("Error associating: %s", err)
	}
	return nil
}
