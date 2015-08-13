package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/objectserver"
	"os"
)

func checkObjectSubcommand(objSrv objectserver.ObjectServer, args []string) {
	hash, err := objectcache.FilenameToHash(args[0])
	if err != nil {
		fmt.Printf("Error parsing hash\t%s\n", err)
		os.Exit(2)
	}
	imageExists, err := objSrv.CheckObject(hash)
	if err != nil {
		fmt.Printf("Error checking object\t%s\n", err)
		os.Exit(2)
	}
	if imageExists {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}
