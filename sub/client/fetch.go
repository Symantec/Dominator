package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func fetch(client *srpc.Client, serverAddress string,
	hashes []hash.Hash) error {
	request := sub.FetchRequest{ServerAddress: serverAddress, Hashes: hashes}
	var reply sub.FetchResponse
	return client.RequestReply("Subd.Fetch", request, &reply)
}
