package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func deleteTagsSubcommand(args []string, logger log.DebugLogger) error {
	if err := deleteTags(args[0], args[1:], logger); err != nil {
		return fmt.Errorf("Error deleting tags for resources: %s", err)
	}
	return nil
}

func deleteTags(tagKey string, resultsFiles []string,
	logger log.DebugLogger) error {
	results := make([]amipublisher.Resource, 0)
	for _, resultsFile := range resultsFiles {
		fileResults := make([]amipublisher.Resource, 0)
		if err := libjson.ReadFromFile(resultsFile, &fileResults); err != nil {
			return err
		}
		results = append(results, fileResults...)
	}
	tagKeys := make([]string, 1)
	tagKeys[0] = tagKey
	return amipublisher.DeleteTags(results, tagKeys, logger)
}

func deleteTagsOnUnpackersSubcommand(args []string,
	logger log.DebugLogger) error {
	if err := deleteTagsOnUnpackers(args[0], logger); err != nil {
		return fmt.Errorf("Error deleting tags on unpackers: %s", err)
	}
	return nil
}

func deleteTagsOnUnpackers(tagKey string, logger log.DebugLogger) error {
	tagKeys := []string{tagKey}
	return amipublisher.DeleteTagsOnUnpackers(targets, skipTargets,
		*instanceName, tagKeys, logger)
}
