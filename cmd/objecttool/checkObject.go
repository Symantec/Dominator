package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/objectserver"
	"os"
)

func checkObjectSubcommand(objSrv objectserver.ObjectServer, args []string) {
	hashes := make([]hash.Hash, 1)
	var err error
	hashes[0], err = objectcache.FilenameToHash(args[0])
	if err != nil {
		fmt.Printf("Error parsing hash\t%s\n", err)
		os.Exit(2)
	}
	objectSizes, err := objSrv.CheckObjects(hashes)
	if err != nil {
		fmt.Printf("Error checking object\t%s\n", err)
		os.Exit(2)
	}
	if objectSizes[0] > 0 {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
