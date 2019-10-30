package main

import (
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func getConfigSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := getConfig(getSubClient()); err != nil {
		logger.Fatalf("Error getting config: %s\n", err)
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
