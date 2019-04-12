package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Symantec/Dominator/imageunpacker/client"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/srpc"
)

func getStatusSubcommand(srpcClient *srpc.Client, args []string) {
	if err := getStatus(srpcClient); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting status: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getStatus(srpcClient *srpc.Client) error {
	status, err := client.GetStatus(srpcClient)
	if err != nil {
		return err
	}
	return json.WriteWithIndent(os.Stdout, "    ", status)
}

func getDeviceForStreamSubcommand(srpcClient *srpc.Client, args []string) {
	if err := getDeviceForStream(srpcClient, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting device for stream: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getDeviceForStream(srpcClient *srpc.Client, streamName string) error {
	status, err := client.GetStatus(srpcClient)
	if err != nil {
		return err
	}
	streamInfo, ok := status.ImageStreams[streamName]
	if !ok {
		return errors.New("unknown stream: " + streamName)
	}
	if streamInfo.DeviceId == "" {
		return errors.New("no device for stream: " + streamName)
	}
	_, err = fmt.Println(streamInfo.DeviceId)
	return err
}
