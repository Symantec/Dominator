package main

import (
	"fmt"
	"io"
	"os"
	"syscall"

	"github.com/Cloud-Foundations/Dominator/lib/fsutil"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
)

const filePerms = syscall.S_IRUSR | syscall.S_IWUSR | syscall.S_IRGRP

func getObjectSubcommand(args []string, logger log.DebugLogger) error {
	hash, err := objectcache.FilenameToHash(args[0])
	if err != nil {
		return fmt.Errorf("Error parsing hash: %s", err)
	}
	if err = getObject(getObjectServer(), hash, args[1]); err != nil {
		return fmt.Errorf("Error getting object: %s", err)
	}
	return nil
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
