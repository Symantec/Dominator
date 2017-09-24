package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/verstr"
	"os"
)

func listStreamsSubcommand(args []string, logger log.Logger) {
	err := listStreams(logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error listing streams: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func listStreams(logger log.Logger) error {
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
