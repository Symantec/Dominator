package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"log"
	"os"
	"path"
	"strings"
	"time"
)

func publishSubcommand(args []string, logger *log.Logger) {
	imageServerAddr := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	err := publish(imageServerAddr, args[0], args[1], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error publishing image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func publish(imageServerAddress string, streamName string, imageLeafName string,
	logger *log.Logger) error {
	streamName = path.Clean(streamName)
	imageLeafName = path.Clean(imageLeafName)
	var tags map[string]string
	if *tagsFile != "" {
		if err := libjson.ReadFromFile(*tagsFile, &tags); err != nil {
			return fmt.Errorf("error loading tags file: %s", err)
		}
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if *expiresIn > 0 {
		expirationTime := time.Now().Add(*expiresIn)
		tags["ExpiresAt"] = expirationTime.UTC().Format("2006-01-02:15:04:05")
	}
	results, err := amipublisher.Publish(imageServerAddress, streamName,
		imageLeafName, *minFreeBytes, *amiName, tags, targets, skipTargets,
		*unpackerName, logger)
	if err != nil {
		return err
	}
	if *ignoreMissingUnpackers {
		newResults := make(amipublisher.Results, 0, len(results))
		for _, result := range results {
			if result.Error != nil &&
				strings.Contains(result.Error.Error(),
					"no ImageUnpacker instances found") {
				continue
			}
			newResults = append(newResults, result)
		}
		results = newResults
	}
	if err := libjson.WriteWithIndent(os.Stdout, "    ", results); err != nil {
		return err
	}
	for _, result := range results {
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}
