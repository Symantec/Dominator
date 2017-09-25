package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
)

func setTagsSubcommand(args []string, logger log.Logger) {
	if err := setTags(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting tags: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func setTags(logger log.Logger) error {
	return amipublisher.SetTags(targets, skipTargets, *instanceName, tags,
		logger)
}
