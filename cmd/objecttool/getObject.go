package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/objectserver"
	"io"
	"os"
)

func getObjectSubcommand(objSrv objectserver.ObjectServer, args []string) {
	hash, err := objectcache.FilenameToHash(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing hash\t%s\n", err)
		os.Exit(2)
	}
	err = getObject(objSrv, hash, args[1])
	if err != nil {
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
	size, reader, err := objectsReader.NextObject()
	if err != nil {
		return err
	}
	defer reader.Close()
	filename := fmt.Sprintf("%s.%x", baseOutputFilename, hashVal)
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	ncopied, err := io.CopyN(file, reader, int64(size))
	if err != nil {
		return err
	}
	if uint64(ncopied) != size {
		return errors.New(fmt.Sprintf("copied: %d, wanted: %d", ncopied, size))
	}
	return nil
}
