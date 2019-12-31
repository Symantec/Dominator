package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func publishSubcommand(args []string, logger log.DebugLogger) error {
	imageServerAddr := fmt.Sprintf("%s:%d",
		*imageServerHostname, *imageServerPortNum)
	if err := publish(imageServerAddr, args[0], args[1], logger); err != nil {
		return fmt.Errorf("Error publishing image: %s\n", err)
	}
	return nil
}

func publish(imageServerAddress string, streamName string, imageLeafName string,
	logger log.DebugLogger) error {
	streamName = path.Clean(streamName)
	imageLeafName = path.Clean(imageLeafName)
	if *expiresIn > 0 {
		expirationTime := time.Now().Add(*expiresIn)
		tags["ExpiresAt"] = expirationTime.UTC().Format(
			amipublisher.ExpiresAtFormat)
	}
	results, err := amipublisher.Publish(imageServerAddress, streamName,
		imageLeafName, *minFreeBytes, *amiName, tags, targets, skipTargets,
		*instanceName, *s3Bucket, *s3Folder, *sharingAccountName,
		amipublisher.PublishOptions{
			EnaSupport: *enaSupport,
		},
		logger)
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
