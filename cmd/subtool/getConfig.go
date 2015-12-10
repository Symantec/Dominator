package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"net/rpc"
	"os"
)

func getConfigSubcommand(client *rpc.Client, args []string) {
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	srpcClient, err := srpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting\t%s\n", err)
		os.Exit(2)
	}
	defer srpcClient.Close()
	if err := getConfig(srpcClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getConfig(srpcClient *srpc.Client) error {
	var request sub.GetConfigurationRequest
	var reply sub.GetConfigurationResponse
	err := client.CallGetConfiguration(srpcClient, request, &reply)
	if err != nil {
		return err
	}
	fmt.Println(reply)
	return nil
}
