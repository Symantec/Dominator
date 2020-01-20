package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func listUnpackersSubcommand(args []string, logger log.DebugLogger) error {
	if err := listUnpackers(logger); err != nil {
		return fmt.Errorf("Error listing unpackers: %s", err)
	}
	return nil
}

func listUnpackers(logger log.DebugLogger) error {
	results, err := amipublisher.ListUnpackers(targets, skipTargets,
		*instanceName, logger)
	if err != nil {
		return err
	}
	return libjson.WriteWithIndent(os.Stdout, "    ", results)
}
