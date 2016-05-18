package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func callSetConfiguration(client *srpc.Client,
	request sub.SetConfigurationRequest,
	reply *sub.SetConfigurationResponse) error {
	return client.RequestReply("Subd.SetConfiguration", request, reply)
}
