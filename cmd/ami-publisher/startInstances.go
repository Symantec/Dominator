package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func startInstancesSubcommand(args []string, logger log.Logger) {
	if err := startInstances(*instanceName, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func startInstances(name string, logger log.Logger) error {
	results, err := amipublisher.StartInstances(targets, skipTargets, name,
		logger)
	if err != nil {
		return err
	}
	if err := libjson.WriteWithIndent(os.Stdout, "    ", results); err != nil {
		return err
	}
	for _, result := range results {
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}
