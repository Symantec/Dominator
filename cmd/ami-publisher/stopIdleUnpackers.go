package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"log"
	"os"
)

func stopIdleUnpackersSubcommand(args []string, logger *log.Logger) {
	err := amipublisher.StopIdleUnpackers(targets, skipTargets, *unpackerName,
		*maxIdleTime, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stopping idle unpackers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
