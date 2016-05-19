package client

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func fetch(client *srpc.Client, serverAddress string,
	hashes []hash.Hash) error {
	request := sub.FetchRequest{serverAddress, hashes}
	var reply sub.FetchResponse
	return client.RequestReply("Subd.Fetch", request, &reply)
}
