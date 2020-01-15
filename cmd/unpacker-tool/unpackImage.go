package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func unpackImageSubcommand(args []string, logger log.DebugLogger) error {
	if err := client.UnpackImage(getClient(), args[0], args[1]); err != nil {
		return fmt.Errorf("Error unpacking image: %s", err)
	}
	return nil
}
