package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func removeUnusedVolumesSubcommand(args []string, logger log.Logger) {
	if err := removeUnusedVolumes(logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing unused volumes: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func removeUnusedVolumes(logger log.Logger) error {
	return amipublisher.RemoveUnusedVolumes(targets, skipTargets, *instanceName,
		logger)
}
