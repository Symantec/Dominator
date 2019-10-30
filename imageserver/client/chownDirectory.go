package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func chownDirectory(client *srpc.Client, dirname, ownerGroup string) error {
	request := imageserver.ChangeOwnerRequest{DirectoryName: dirname,
		OwnerGroup: ownerGroup}
	var reply imageserver.ChangeOwnerResponse
	return client.RequestReply("ImageServer.ChownDirectory", request, &reply)
}
