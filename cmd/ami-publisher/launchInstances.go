package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/awsutil"
	"github.com/Symantec/Dominator/lib/log"
	"os"
	"path"
)

func launchInstancesSubcommand(args []string, logger log.Logger) {
	domImage := ""
	if len(args) > 1 {
		domImage = args[1]
	}
	err := launchInstances(args[0], domImage, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error launching instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func launchInstances(bootImage, domImage string, logger log.Logger) error {
	bootImage = path.Clean(bootImage)
	tags, err := makeTags()
	if err != nil {
		return err
	}
	tags["Name"] = *unpackerName
	if domImage != "" {
		tags["RequiredImage"] = domImage
	}
	imageTags := make(awsutil.Tags)
	for key, value := range searchTags {
		imageTags[key] = value
	}
	imageTags["Name"] = bootImage
	return amipublisher.LaunchInstances(targets, skipTargets, imageTags,
		vpcSearchTags, subnetSearchTags, securityGroupSearchTags,
		*instanceType, *sshKeyName, tags, logger)
}
