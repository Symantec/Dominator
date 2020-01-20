package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listUsedImagesSubcommand(args []string, logger log.DebugLogger) error {
	if err := listUsedImages(logger); err != nil {
		return fmt.Errorf("Error listing used images: %s", err)
	}
	logMemoryUsage(logger)
	return nil
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
