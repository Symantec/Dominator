package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func setTagsSubcommand(args []string, logger log.Logger) {
	err := setTags(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting tags: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func setTags(logger log.Logger) error {
	tags, err := makeTags()
	if err != nil {
		return err
	}
	return amipublisher.SetTags(targets, skipTargets, *instanceName, tags,
		logger)
}
