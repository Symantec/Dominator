package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"log"
	"os"
)

func setExclusiveTagsSubcommand(args []string, logger *log.Logger) {
	err := setExclusiveTags(args[0], args[1], args[2:], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr,
			"Error setting exclusive tag for resources: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func setExclusiveTags(tagKey string, tagValue string, resultsFiles []string,
	logger *log.Logger) error {
	results := make([]amipublisher.Resource, 0)
	for _, resultsFile := range resultsFiles {
		fileResults := make([]amipublisher.Resource, 0)
		if err := libjson.ReadFromFile(resultsFile, &fileResults); err != nil {
			return err
		}
		results = append(results, fileResults...)
	}
	if tagKey == "" {
		return errors.New("empty tag key specified")
	}
	if tagKey == "Name" {
		return errors.New("cannot set exclusive Name tag")
	}
	return amipublisher.SetExclusiveTags(results, tagKey, tagValue, logger)
}
