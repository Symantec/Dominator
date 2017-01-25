package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
	"os"
	"path"
)

func launchInstancesSubcommand(args []string, logger log.Logger) {
	if err := launchInstances(args[0], logger); err != nil {
		fmt.Fprintf(os.Stderr, "Error launching instances: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func launchInstances(bootImage string, logger log.Logger) error {
	bootImage = path.Clean(bootImage)
	tags["Name"] = *instanceName
	searchTags["Name"] = bootImage
	return amipublisher.LaunchInstances(targets, skipTargets, searchTags,
		vpcSearchTags, subnetSearchTags, securityGroupSearchTags,
		*instanceType, *sshKeyName, tags, logger)
}
