package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"github.com/Symantec/Dominator/lib/log"
	"os"
	"path"
)

func copyBootstrapImageSubcommand(args []string, logger log.Logger) {
	err := copyBootstrapImage(args[0], logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying bootstrap image: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func copyBootstrapImage(streamName string, logger log.Logger) error {
	streamName = path.Clean(streamName)
	tags["Name"] = streamName
	return amipublisher.CopyBootstrapImage(streamName, targets, skipTargets,
		*marketplaceImage, *marketplaceLoginName, tags, *instanceName,
		vpcSearchTags, subnetSearchTags, securityGroupSearchTags, *instanceType,
		*sshKeyName, logger)
}
