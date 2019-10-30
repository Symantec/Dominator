package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
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
