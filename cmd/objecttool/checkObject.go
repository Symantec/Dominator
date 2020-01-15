package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
)

func checkObjectSubcommand(args []string, logger log.DebugLogger) error {
	hashes := make([]hash.Hash, 1)
	var err error
	hashes[0], err = objectcache.FilenameToHash(args[0])
	if err != nil {
		return fmt.Errorf("Error parsing hash: %s", err)
	}
	objectSizes, err := getObjectServer().CheckObjects(hashes)
	if err != nil {
		return fmt.Errorf("Error checking object: %s", err)
	}
	if objectSizes[0] > 0 {
		return nil
	} else {
		os.Exit(1)
		panic("impossible")
	}
}
