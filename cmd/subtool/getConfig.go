package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func getConfigSubcommand(srpcClient *srpc.Client, args []string) {
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
