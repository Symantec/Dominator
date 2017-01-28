package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func terminateInstancesSubcommand(args []string, logger log.Logger) {
	err := amipublisher.TerminateInstances(targets, skipTargets, *instanceName,
		logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error terminating instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
