package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func CallAddImage(client *srpc.Client, request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	return callAddImage(client, request, reply)
}

func CallDeleteImage(client *srpc.Client,
	request imageserver.DeleteImageRequest,
	reply *imageserver.DeleteImageResponse) error {
	return callDeleteImage(client, request, reply)
}

func CallGetImage(client *srpc.Client, request imageserver.GetImageRequest,
	reply *imageserver.GetImageResponse) error {
	return callGetImage(client, request, reply)
}
