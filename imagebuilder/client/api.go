package client

import (
	"io"

	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imaginator"
)

func BuildImage(client *srpc.Client, request proto.BuildImageRequest,
	response *proto.BuildImageResponse, logWriter io.Writer) error {
	return buildImage(client, request, response, logWriter)
}
