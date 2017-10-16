package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
)

func addVolumesSubcommand(args []string, logger log.DebugLogger) {
	if err := addVolumes(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding volumes: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func addVolumes(sizeStr string, logger log.DebugLogger) error {
	sizeInGiB, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		return err
	}
	return amipublisher.AddVolumes(targets, skipTargets, tags,
		*instanceName, uint64(sizeInGiB)<<30, logger)
}
