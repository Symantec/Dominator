package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func callDeleteImage(client *srpc.Client, name string) error {
	request := imageserver.DeleteImageRequest{name}
	var reply imageserver.DeleteImageResponse
	return client.RequestReply("ImageServer.DeleteImage", request, &reply)
}
