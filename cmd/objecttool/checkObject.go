package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

func checkObjectSubcommand(objSrv objectserver.ObjectServer, args []string) {
	hashes := make([]hash.Hash, 1)
	var err error
	hashes[0], err = objectcache.FilenameToHash(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing hash\t%s\n", err)
		os.Exit(2)
	}
	objectSizes, err := objSrv.CheckObjects(hashes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking object\t%s\n", err)
		os.Exit(2)
	}
	if objectSizes[0] > 0 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
