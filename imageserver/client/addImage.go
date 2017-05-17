package client

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func addImage(client *srpc.Client, name string, img *image.Image) error {
	request := imageserver.AddImageRequest{name, img}
	var reply imageserver.AddImageResponse
	return client.RequestReply("ImageServer.AddImage", request, &reply)
}

func addImageTrusted(client *srpc.Client, name string, img *image.Image) error {
	request := imageserver.AddImageRequest{name, img}
	var reply imageserver.AddImageResponse
	return client.RequestReply("ImageServer.AddImageTrusted", request, &reply)
}
