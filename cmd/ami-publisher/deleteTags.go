package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func deleteTagsSubcommand(args []string, logger log.Logger) {
	err := deleteTags(args[0], args[1:], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting tags for resources: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func deleteTags(tagKey string, resultsFiles []string, logger log.Logger) error {
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
