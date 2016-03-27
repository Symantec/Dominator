package main

import (
	"errors"
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
	var request sub.SetConfigurationRequest
	request.ScanSpeedPercent = *scanSpeedPercent
	request.NetworkSpeedPercent = *networkSpeedPercent
	request.ScanExclusionList = scanExcludeList
	var reply sub.SetConfigurationResponse
	err := client.CallSetConfiguration(srpcClient, request, &reply)
	if err != nil {
		return err
	}
	if reply.Success {
		return nil
	}
	return errors.New("Error setting configuration: " + reply.ErrorString)
}
