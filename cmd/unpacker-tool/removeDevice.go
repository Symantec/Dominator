package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func removeDeviceSubcommand(args []string, logger log.DebugLogger) error {
	if err := client.RemoveDevice(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error removing device: %s", err)
	}
	return nil
}
