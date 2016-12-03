package main

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
)

func getStatusSubcommand(client *srpc.Client, args []string) {
	if err := getStatus(client); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting status: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getStatus(client *srpc.Client) error {
	var request proto.GetStatusRequest
	var reply proto.GetStatusResponse
	err := client.RequestReply("ImageUnpacker.GetStatus", request, &reply)
	if err != nil {
		return err
	}
	return json.WriteWithIndent(os.Stdout, "    ", reply)
}

func getDeviceForStreamSubcommand(client *srpc.Client, args []string) {
	if err := getDeviceForStream(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error getting device for stream: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func getDeviceForStream(client *srpc.Client, streamName string) error {
	var request proto.GetStatusRequest
	var reply proto.GetStatusResponse
	err := client.RequestReply("ImageUnpacker.GetStatus", request, &reply)
	if err != nil {
		return err
	}
	streamInfo, ok := reply.ImageStreams[streamName]
	if !ok {
		return errors.New("unknown stream: " + streamName)
	}
	if streamInfo.DeviceId == "" {
		return errors.New("no device for stream: " + streamName)
	}
	_, err = fmt.Println(streamInfo.DeviceId)
	return err
}
