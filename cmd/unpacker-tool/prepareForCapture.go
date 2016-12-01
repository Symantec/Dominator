package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
)

func prepareForCaptureSubcommand(client *srpc.Client, args []string) {
	if err := prepareForCapture(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing for capture: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func prepareForCapture(client *srpc.Client, streamName string) error {
	request := proto.PrepareForCaptureRequest{streamName}
	var reply proto.PrepareForCaptureResponse
	return client.RequestReply("ImageUnpacker.PrepareForCapture", request,
		&reply)
}
