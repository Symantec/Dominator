package main

import (
	"fmt"
	"github.com/Symantec/Dominator/imagepublishers/amipublisher"
	"log"
	"os"
)

func expireSubcommand(args []string, logger *log.Logger) {
	err := amipublisher.ExpireResources(targets, skipTargets, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error expiring resources: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}
