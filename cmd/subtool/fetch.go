package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func fetchSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := fetch(getSubClient(), args[0]); err != nil {
		logger.Fatalf("Error fetching: %s\n", err)
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
	return srpcClient.RequestReply("Subd.Fetch", sub.FetchRequest{
		ServerAddress: serverAddress,
		Wait:          true,
		Hashes:        hashes},
		&sub.FetchResponse{})
}
