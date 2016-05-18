package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"os"
)

func setConfigSubcommand(srpcClient *srpc.Client, args []string) {
	if err := setConfig(srpcClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error setting config\t%s\n", err)
		os.Exit(1)
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
