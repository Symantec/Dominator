package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/Cloud-Foundations/Dominator/imageunpacker/client"
	"github.com/Cloud-Foundations/Dominator/lib/json"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func getStatusSubcommand(args []string, logger log.DebugLogger) error {
	if err := getStatus(getClient()); err != nil {
		return fmt.Errorf("Error getting status: %s", err)
	}
	return nil
}

func getStatus(srpcClient *srpc.Client) error {
	status, err := client.GetStatus(srpcClient)
	if err != nil {
		return err
	}
	return json.WriteWithIndent(os.Stdout, "    ", status)
}

func getDeviceForStreamSubcommand(args []string, logger log.DebugLogger) error {
	if err := getDeviceForStream(getClient(), args[0]); err != nil {
		return fmt.Errorf("Error getting device for stream: %s", err)
	}
	return nil
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
