package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	libjson "github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/log"
	libtags "github.com/Symantec/Dominator/lib/tags"
)

func launchInstancesSubcommand(args []string, logger log.DebugLogger) {
	if err := launchInstances(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func launchInstances(bootImage string, logger log.DebugLogger) error {
	bootImage = path.Clean(bootImage)
	tags["Name"] = *instanceName
	searchTags["Name"] = bootImage
	results, err := amipublisher.LaunchInstances(targets, skipTargets,
		searchTags, vpcSearchTags, subnetSearchTags, securityGroupSearchTags,
		*instanceType, *sshKeyName, tags, *replaceInstances, logger)
	if err != nil {
		return err
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

func launchInstancesForImagesSubcommand(args []string, logger log.DebugLogger) {
	if err := launchInstancesForImages(args, logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func launchInstancesForImages(resourcesFiles []string,
	logger log.DebugLogger) error {
	resources := make([]amipublisher.Resource, 0)
	for _, resourcesFile := range resourcesFiles {
		fileRes := make([]amipublisher.Resource, 0)
		if err := libjson.ReadFromFile(resourcesFile, &fileRes); err != nil {
			return err
		}
		resources = append(resources, fileRes...)
	}
	if tags == nil {
		tags = make(libtags.Tags)
	}
	tags["Name"] = *instanceName
	if *expiresIn > 0 {
		expirationTime := time.Now().Add(*expiresIn)
		tags["ExpiresAt"] = expirationTime.UTC().Format(
			amipublisher.ExpiresAtFormat)
	}
	results, err := amipublisher.LaunchInstancesForImages(resources,
		vpcSearchTags, subnetSearchTags, securityGroupSearchTags,
		*instanceType, *sshKeyName, tags, logger)
	if err != nil {
		return err
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
