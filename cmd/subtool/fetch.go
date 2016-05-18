package main

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func fetchSubcommand(srpcClient *srpc.Client, args []string) {
	if err := fetch(srpcClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching\t%s\n", err)
		os.Exit(2)
	}
	os.Exit(0)
}

func fetch(srpcClient *srpc.Client, hashesFilename string) error {
	hashesFile, err := os.Open(hashesFilename)
	if err != nil {
		return err
	}
	defer hashesFile.Close()
	scanner := bufio.NewScanner(hashesFile)
	serverAddress := fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum)
	hashes := make([]hash.Hash, 0)
	for scanner.Scan() {
		hashval, err := objectcache.FilenameToHash(scanner.Text())
		if err != nil {
			return err
		}
		hashes = append(hashes, hashval)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return client.Fetch(srpcClient, serverAddress, hashes)
}
