package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/sub"
	"net/rpc"
	"os"
	"strings"
)

func setConfigSubcommand(client *rpc.Client, args []string) {
	err := setConfig(client)
	if err != nil {
		fmt.Printf("Error setting config\t%s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func setConfig(client *rpc.Client) error {
	var request sub.SetConfigurationRequest
	request.ScanSpeedPercent = *scanSpeedPercent
	request.NetworkSpeedPercent = *networkSpeedPercent
	request.ScanExclusionList = strings.Split(*scanExcludeList, ",")
	var reply sub.SetConfigurationResponse
	err := client.Call("Subd.SetConfiguration", request, &reply)
	if err != nil {
		return err
	}
	if reply.Success {
		return nil
	}
	return errors.New("Error setting configuration: " + reply.ErrorString)
}
