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
		fmt.Printf("Error parsing hash\t%s\n", err)
		os.Exit(2)
	}
	err = getObject(objSrv, hash, args[1])
	if err != nil {
		fmt.Printf("Error getting object\t%s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func getObject(objSrv objectserver.ObjectServer, hash hash.Hash,
	baseOutputFilename string) error {
	size, reader, err := objSrv.GetObjectReader(hash)
	if err != nil {
		return err
	}
	defer reader.Close()
	filename := fmt.Sprintf("%s.%x", baseOutputFilename, hash)
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
