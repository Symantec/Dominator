package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listUsedImagesSubcommand(args []string, logger log.DebugLogger) {
	err := listUsedImages(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing used images: %s\n", err)
		os.Exit(1)
	}
	logMemoryUsage(logger)
	os.Exit(0)
}

func listUsedImages(logger log.DebugLogger) error {
	results, err := amipublisher.ListUsedImages(targets, skipTargets,
		searchTags, excludeSearchTags, logger)
	if err != nil {
		return err
	}
	if err := libjson.WriteWithIndent(os.Stdout, "    ", results); err != nil {
		return err
	}
	return nil
}
