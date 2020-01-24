package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func terminateInstancesSubcommand(args []string, logger log.DebugLogger) error {
	err := amipublisher.TerminateInstances(targets, skipTargets, *instanceName,
		logger)
	if err != nil {
		return fmt.Errorf("Error terminating instances: %s", err)
	}
	return nil
}
