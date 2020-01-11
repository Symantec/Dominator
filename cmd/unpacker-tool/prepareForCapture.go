package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func prepareForCaptureSubcommand(args []string, logger log.DebugLogger) error {
	if err := client.PrepareForCapture(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error preparing for capture: %s", err)
	}
	return nil
}
