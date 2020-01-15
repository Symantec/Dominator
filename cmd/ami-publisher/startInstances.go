package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func startInstancesSubcommand(args []string, logger log.DebugLogger) error {
	if err := startInstances(*instanceName, logger); err != nil {
		return fmt.Errorf("Error starting instances: %s\n", err)
	}
	return nil
}

func startInstances(name string, logger log.DebugLogger) error {
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
