package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/imagepublishers/amipublisher"
	"github.com/Cloud-Foundations/Dominator/lib/log"
)

func expireSubcommand(args []string, logger log.DebugLogger) error {
	err := amipublisher.ExpireResources(targets, skipTargets, logger)
	if err != nil {
		return fmt.Errorf("Error expiring resources: %s\n", err)
	}
	return nil
}
