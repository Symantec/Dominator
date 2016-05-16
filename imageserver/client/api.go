package client

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func CallAddImage(client *srpc.Client, request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	return callAddImage(client, request, reply)
}

func CallCheckImage(client *srpc.Client, name string) (bool, error) {
	return callCheckImage(client, name)
}

func CallChownDirectory(client *srpc.Client, dirname, ownerGroup string) error {
	return callChownDirectory(client, dirname, ownerGroup)
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

func CallListDirectories(client *srpc.Client) ([]image.Directory, error) {
	return callListDirectories(client)
}

func CallListImages(client *srpc.Client) ([]string, error) {
	return callListImages(client)
}

func CallMakeDirectory(client *srpc.Client, dirname string) error {
	return callMakeDirectory(client, dirname)
}
