package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func callGetConfiguration(client *srpc.Client,
	request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	return client.RequestReply("Subd.GetConfiguration", request, reply)
}
