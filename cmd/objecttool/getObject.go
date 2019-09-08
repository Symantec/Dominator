package main

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/objectserver"
)

const filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP

func getObjectSubcommand(objSrv objectserver.ObjectServer, args []string) {
	hash, err := objectcache.FilenameToHash(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing hash\t%s\n", err)
		os.Exit(2)
	}
	if err = getObject(objSrv, hash, args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting object\t%s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func getObject(objSrv objectserver.ObjectServer, hashVal hash.Hash,
	baseOutputFilename string) error {
	hashes := make([]hash.Hash, 1)
	hashes[0] = hashVal
	objectsReader, err := objSrv.GetObjects(hashes)
	if err != nil {
		return err
	}
	defer objectsReader.Close()
	size, reader, err := objectsReader.NextObject()
	if err != nil {
		return err
	}
	defer reader.Close()
	if baseOutputFilename == "-" {
		_, err := io.Copy(os.Stdout, reader)
		return err
	} else {
		filename := fmt.Sprintf("%s.%x", baseOutputFilename, hashVal)
		return fsutil.CopyToFile(filename, filePerms, reader, size)
	}
}
