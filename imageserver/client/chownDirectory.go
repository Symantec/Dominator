package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func callChownDirectory(client *srpc.Client, dirname, ownerGroup string) error {
	request := imageserver.ChangeOwnerRequest{DirectoryName: dirname,
		OwnerGroup: ownerGroup}
	var reply imageserver.ChangeOwnerResponse
	return client.RequestReply("ImageServer.ChownDirectory", request, &reply)
}
