package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
	"os"
	"path"
)

func prepareUnpackersSubcommand(args []string, logger log.Logger) {
	streamName := ""
	if len(args) > 0 {
		streamName = path.Clean(args[0])
	}
	err := amipublisher.PrepareUnpackers(streamName, targets, skipTargets,
		*instanceName, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing unpackers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
