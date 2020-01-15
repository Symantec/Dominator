package main

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/Dominator/sub/client"
)

func setConfigSubcommand(args []string, logger log.DebugLogger) error {
	srpcClient := getSubClient(logger)
	defer srpcClient.Close()
	if err := setConfig(srpcClient); err != nil {
		return fmt.Errorf("Error setting config: %s", err)
	}
	return nil
}

func setConfig(srpcClient *srpc.Client) error {
	var config sub.Configuration
	config.CpuPercent = *cpuPercent
	config.NetworkSpeedPercent = *networkSpeedPercent
	config.ScanExclusionList = scanExcludeList
	config.ScanSpeedPercent = *scanSpeedPercent
	return client.SetConfiguration(srpcClient, config)
}
