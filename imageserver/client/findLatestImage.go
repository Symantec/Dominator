package client

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func findLatestImage(client *srpc.Client, dirname string,
	ignoreExpiring bool) (string, error) {
	request := imageserver.FindLatestImageRequest{
		DirectoryName:        dirname,
		IgnoreExpiringImages: ignoreExpiring,
	}
	var reply imageserver.FindLatestImageResponse
	err := client.RequestReply("ImageServer.FindLatestImage", request, &reply)
	if err == nil {
		err = errors.New(reply.Error)
	}
	if err != nil {
		return "", err
	}
	return reply.ImageName, nil
}
