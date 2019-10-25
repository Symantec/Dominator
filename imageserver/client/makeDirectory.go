package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func makeDirectory(client *srpc.Client, dirname string) error {
	request := imageserver.MakeDirectoryRequest{dirname}
	var reply imageserver.MakeDirectoryResponse
	return client.RequestReply("ImageServer.MakeDirectory", request, &reply)
}
