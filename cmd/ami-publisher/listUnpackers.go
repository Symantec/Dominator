package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"os"
)

func listUnpackersSubcommand(args []string, logger log.Logger) {
	err := listUnpackers(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing unpackers: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listUnpackers(logger log.Logger) error {
	results, err := amipublisher.ListUnpackers(targets, skipTargets,
		*unpackerName, logger)
	if err != nil {
		return err
	}
	return libjson.WriteWithIndent(os.Stdout, "    ", results)
}
