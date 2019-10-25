package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func callUpdate(client *srpc.Client, request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	return client.RequestReply("Subd.Update", request, reply)
}
