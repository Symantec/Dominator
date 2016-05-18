package client

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func cleanup(client *srpc.Client, hashes []hash.Hash) error {
	request := sub.CleanupRequest{hashes}
	var reply sub.CleanupResponse
	return client.RequestReply("Subd.Cleanup", request, &reply)
}
