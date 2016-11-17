package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func getConfigSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := getConfig(getSubClient()); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getConfig(srpcClient *srpc.Client) error {
	config, err := client.GetConfiguration(srpcClient)
	if err != nil {
		return err
	}
	fmt.Println(config)
	return nil
}
