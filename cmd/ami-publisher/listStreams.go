package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/verstr"
)

func listStreamsSubcommand(args []string, logger log.DebugLogger) {
	err := listStreams(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing streams: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listStreams(logger log.DebugLogger) error {
	results, err := amipublisher.ListStreams(targets, skipTargets,
		*instanceName, logger)
	if err != nil {
		return err
	}
	streams := make([]string, 0, len(results))
	for stream := range results {
		streams = append(streams, stream)
	}
	verstr.Sort(streams)
	return libjson.WriteWithIndent(os.Stdout, "    ", streams)
}
