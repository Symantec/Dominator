package client

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"time"
)

func getImage(client *srpc.Client, name string, timeout time.Duration) (
	*image.Image, error) {
	request := imageserver.GetImageRequest{ImageName: name, Timeout: timeout}
	var reply imageserver.GetImageResponse
	err := client.RequestReply("ImageServer.GetImage", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Image, nil
}
