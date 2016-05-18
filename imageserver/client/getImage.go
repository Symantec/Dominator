package client

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func callGetImage(client *srpc.Client, name string) (*image.Image, error) {
	request := imageserver.GetImageRequest{name}
	var reply imageserver.GetImageResponse
	err := client.RequestReply("ImageServer.GetImage", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Image, nil
}
