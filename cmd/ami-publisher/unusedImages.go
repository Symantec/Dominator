package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
)

func listUnusedImagesSubcommand(args []string, logger log.DebugLogger) {
	err := listUnusedImages(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing unused images: %s\n", err)
		os.Exit(1)
	}
	logMemoryUsage(logger)
	os.Exit(0)
}

func listUnusedImages(logger log.DebugLogger) error {
	results, err := amipublisher.ListUnusedImages(targets, skipTargets,
		searchTags, excludeSearchTags, *minImageAge, logger)
	if err != nil {
		return err
	}
	return libjson.WriteWithIndent(os.Stdout, "    ", results)
}

func deleteUnusedImagesSubcommand(args []string, logger log.DebugLogger) {
	err := deleteUnusedImages(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting unused images: %s\n", err)
		os.Exit(1)
	}
	logMemoryUsage(logger)
	os.Exit(0)
}

func deleteUnusedImages(logger log.DebugLogger) error {
	results, err := amipublisher.DeleteUnusedImages(targets, skipTargets,
		searchTags, excludeSearchTags, *minImageAge, logger)
	if err != nil {
		return err
	}
	return libjson.WriteWithIndent(os.Stdout, "    ", results)
}
