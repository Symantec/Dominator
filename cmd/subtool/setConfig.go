package main

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func setConfigSubcommand(getSubClient getSubClientFunc, args []string) {
	if err := setConfig(getSubClient()); err != nil {
		logger.Fatalf("Error setting config: %s\n", err)
	}
	os.Exit(0)
}

func setConfig(srpcClient *srpc.Client) error {
	var config sub.Configuration
	config.ScanSpeedPercent = *scanSpeedPercent
	config.NetworkSpeedPercent = *networkSpeedPercent
	config.ScanExclusionList = scanExcludeList
	return client.SetConfiguration(srpcClient, config)
}
