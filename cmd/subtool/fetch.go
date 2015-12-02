package main

import (
	"bufio"
	"fmt"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"net/rpc"
	"os"
)

func fetchSubcommand(rpcClient *rpc.Client, args []string) {
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	srpcClient, err := srpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting\t%s\n", err)
		os.Exit(2)
	}
	defer srpcClient.Close()
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
	var request sub.FetchRequest
	var reply sub.FetchResponse
	request.ServerAddress = fmt.Sprintf("%s:%d",
		*objectServerHostname, *objectServerPortNum)
	for scanner.Scan() {
		hashval, err := objectcache.FilenameToHash(scanner.Text())
		if err != nil {
			return err
		}
		request.Hashes = append(request.Hashes, hashval)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return client.CallFetch(srpcClient, request, &reply)
}
