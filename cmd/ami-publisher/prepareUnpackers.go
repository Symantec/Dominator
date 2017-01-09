package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"log"
	"os"
	"path"
)

func prepareUnpackersSubcommand(args []string, logger *log.Logger) {
	err := prepareUnpackers(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing unpackers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func prepareUnpackers(streamName string, logger *log.Logger) error {
	streamName = path.Clean(streamName)
	return amipublisher.PrepareUnpackers(streamName, targets, skipTargets,
		*unpackerName, logger)
}
