package main

import (
	"fmt"
	"path"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func prepareUnpackersSubcommand(args []string, logger log.DebugLogger) error {
	streamName := ""
	if len(args) > 0 {
		streamName = path.Clean(args[0])
	}
	err := amipublisher.PrepareUnpackers(streamName, targets, skipTargets,
		*instanceName, logger)
	if err != nil {
		return fmt.Errorf("Error preparing unpackers: %s\n", err)
	}
	return nil
}
