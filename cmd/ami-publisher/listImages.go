package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listImagesSubcommand(args []string, logger log.DebugLogger) error {
	if err := listImages(logger); err != nil {
		return fmt.Errorf("Error listing images: %s\n", err)
	}
	return nil
}

func listImages(logger log.DebugLogger) error {
	results, err := amipublisher.ListImages(targets, skipTargets, searchTags,
		excludeSearchTags, *minImageAge, logger)
	if err != nil {
		return err
	}
	return libjson.WriteWithIndent(os.Stdout, "    ", results)
}
