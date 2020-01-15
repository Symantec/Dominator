package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func fetchSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := fetch(srpcClient, args[0]); err != nil {
		return fmt.Errorf("Error fetching: %s", err)
	}
	return nil
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
