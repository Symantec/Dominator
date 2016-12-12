package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
)

func associateSubcommand(client *srpc.Client, args []string) {
	if err := associate(client, args[0], args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error associating: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func associate(client *srpc.Client, streamName string, deviceId string) error {
	request := proto.AssociateStreamWithDeviceRequest{streamName, deviceId}
	var reply proto.AssociateStreamWithDeviceResponse
	return client.RequestReply("ImageUnpacker.AssociateStreamWithDevice",
		request, &reply)
}
