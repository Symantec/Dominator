package main

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/imaginator"
	"os"
)

func buildImageSubcommand(args []string, logger log.Logger) {
	srpcClient := getImaginatorClient()
	request := proto.BuildImageRequest{
		StreamName: args[0],
		ExpiresIn:  *expiresIn,
	}
	if len(args) > 1 {
		request.GitBranch = args[1]
	}
	var reply proto.BuildImageResponse
	err := srpcClient.RequestReply("Imaginator.BuildImage", request, &reply)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error building image: %s\n", err)
		os.Exit(1)
	}
	if reply.ErrorString != "" {
		fmt.Fprintf(os.Stderr, "Error building image: %s\n", reply.ErrorString)
		os.Stderr.Write(reply.BuildLog)
		os.Exit(1)
	}
	fmt.Println(reply.ImageName)
	os.Exit(0)
}
