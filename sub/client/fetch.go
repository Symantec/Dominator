package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func callFetch(client *srpc.Client, request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	return client.RequestReply("Subd.Fetch", request, reply)
}
