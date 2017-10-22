package main

import (
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
)

func listUnpackersSubcommand(args []string, logger log.DebugLogger) {
	err := listUnpackers(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing unpackers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listUnpackers(logger log.DebugLogger) error {
	results, err := amipublisher.ListUnpackers(targets, skipTargets,
		*instanceName, logger)
	if err != nil {
		return err
	}
	return libjson.WriteWithIndent(os.Stdout, "    ", results)
}
