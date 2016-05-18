package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func callUpdate(client *srpc.Client, request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	return client.RequestReply("Subd.Update", request, reply)
}
