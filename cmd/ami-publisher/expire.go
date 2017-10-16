package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
)

func expireSubcommand(args []string, logger log.DebugLogger) {
	err := amipublisher.ExpireResources(targets, skipTargets, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expiring resources: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
