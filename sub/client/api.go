package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func CallFetch(client *srpc.Client, request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	return callFetch(client, request, reply)
}
