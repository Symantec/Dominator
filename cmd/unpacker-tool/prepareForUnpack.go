package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
	"os"
)

func prepareForUnpackSubcommand(client *srpc.Client, args []string) {
	if err := prepareForUnpack(client, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "Error preparing for unpack: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func prepareForUnpack(client *srpc.Client, streamName string) error {
	request := proto.PrepareForUnpackRequest{StreamName: streamName}
	var reply proto.PrepareForUnpackResponse
	return client.RequestReply("ImageUnpacker.PrepareForUnpack", request,
		&reply)
}
