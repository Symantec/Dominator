package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func makeDirectory(client *srpc.Client, dirname string) error {
	request := imageserver.MakeDirectoryRequest{dirname}
	var reply imageserver.MakeDirectoryResponse
	return client.RequestReply("ImageServer.MakeDirectory", request, &reply)
}
