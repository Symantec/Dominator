package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func CallAddImage(client *srpc.Client, request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	return callAddImage(client, request, reply)
}
