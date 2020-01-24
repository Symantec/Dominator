package main

import (
	"fmt"
	"path"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func copyBootstrapImageSubcommand(args []string, logger log.DebugLogger) error {
	if err := copyBootstrapImage(args[0], logger); err != nil {
		return fmt.Errorf("Error copying bootstrap image: %s", err)
	}
	return nil
}

func copyBootstrapImage(streamName string, logger log.DebugLogger) error {
	streamName = path.Clean(streamName)
	tags["Name"] = streamName
	return amipublisher.CopyBootstrapImage(streamName, targets, skipTargets,
		*marketplaceImage, *marketplaceLoginName, tags, *instanceName,
		vpcSearchTags, subnetSearchTags, securityGroupSearchTags, *instanceType,
		*sshKeyName, logger)
}
