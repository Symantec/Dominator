package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func callCleanup(client *srpc.Client, request sub.CleanupRequest,
	reply *sub.CleanupResponse) error {
	return client.RequestReply("Subd.Cleanup", request, reply)
}
