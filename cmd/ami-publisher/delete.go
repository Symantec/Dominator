package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func deleteSubcommand(args []string, logger log.Logger) {
	err := deleteResources(args, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error deleting resources: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func deleteResources(resultsFiles []string, logger log.Logger) error {
	results := make([]amipublisher.Resource, 0)
	for _, resultsFile := range resultsFiles {
		fileResults := make([]amipublisher.Resource, 0)
		if err := libjson.ReadFromFile(resultsFile, &fileResults); err != nil {
			return err
		}
		results = append(results, fileResults...)
	}
	return amipublisher.DeleteResources(results, logger)
}
