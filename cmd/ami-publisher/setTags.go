package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func setTagsSubcommand(args []string, logger log.DebugLogger) error {
	if err := setTags(logger); err != nil {
		return fmt.Errorf("Error setting tags: %s\n", err)
	}
	return nil
}

func setTags(logger log.DebugLogger) error {
	return amipublisher.SetTags(targets, skipTargets, *instanceName, tags,
		logger)
}
