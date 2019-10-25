package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func deleteImage(client *srpc.Client, name string) error {
	request := imageserver.DeleteImageRequest{name}
	var reply imageserver.DeleteImageResponse
	return client.RequestReply("ImageServer.DeleteImage", request, &reply)
}
