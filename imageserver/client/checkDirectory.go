package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func checkDirectory(client *srpc.Client, name string) (bool, error) {
	request := imageserver.CheckDirectoryRequest{name}
	var reply imageserver.CheckDirectoryResponse
	err := client.RequestReply("ImageServer.CheckDirectory", request, &reply)
	if err != nil {
		return false, err
	}
	return reply.DirectoryExists, nil
}
