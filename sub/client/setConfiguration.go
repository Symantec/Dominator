package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func setConfiguration(client *srpc.Client, config sub.Configuration) error {
	var request sub.SetConfigurationRequest
	request = sub.SetConfigurationRequest(config)
	var reply sub.SetConfigurationResponse
	return client.RequestReply("Subd.SetConfiguration", request, &reply)
}
