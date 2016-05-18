package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func getConfiguration(client *srpc.Client) (sub.Configuration, error) {
	var request sub.GetConfigurationRequest
	var reply sub.GetConfigurationResponse
	err := client.RequestReply("Subd.GetConfiguration", request, &reply)
	return sub.Configuration(reply), err
}
