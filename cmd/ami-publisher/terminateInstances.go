package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func terminateInstancesSubcommand(args []string, logger log.DebugLogger) {
	err := amipublisher.TerminateInstances(targets, skipTargets, *instanceName,
		logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error terminating instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
