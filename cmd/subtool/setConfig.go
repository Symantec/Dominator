package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"github.com/Symantec/Dominator/sub/client"
	"net/rpc"
	"os"
	"strings"
)

func setConfigSubcommand(client *rpc.Client, args []string) {
	clientName := fmt.Sprintf("%s:%d", *subHostname, *subPortNum)
	srpcClient, err := srpc.DialHTTP("tcp", clientName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting\t%s\n", err)
		os.Exit(2)
	}
	defer srpcClient.Close()
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
	request.ScanExclusionList = strings.Split(*scanExcludeList, ",")
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
