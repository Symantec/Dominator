package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func getConfigSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := getConfig(srpcClient); err != nil {
		return fmt.Errorf("Error getting config: %s", err)
	}
	return nil
}

func getConfig(srpcClient *srpc.Client) error {
	config, err := client.GetConfiguration(srpcClient)
	if err != nil {
		return err
	}
	fmt.Println(config)
	return nil
}
