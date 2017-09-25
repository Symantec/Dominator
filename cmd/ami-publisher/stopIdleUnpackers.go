package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
)

func stopIdleUnpackersSubcommand(args []string, logger log.Logger) {
	err := amipublisher.StopIdleUnpackers(targets, skipTargets, *instanceName,
		*maxIdleTime, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping idle unpackers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
