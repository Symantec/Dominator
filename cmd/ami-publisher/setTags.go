package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func setTagsSubcommand(args []string, logger log.DebugLogger) {
	if err := setTags(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting tags: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func setTags(logger log.DebugLogger) error {
	return amipublisher.SetTags(targets, skipTargets, *instanceName, tags,
		logger)
}
