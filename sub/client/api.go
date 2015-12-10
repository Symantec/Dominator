package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func CallFetch(client *srpc.Client, request sub.FetchRequest,
	reply *sub.FetchResponse) error {
	return callFetch(client, request, reply)
}

func CallGetConfiguration(client *srpc.Client,
	request sub.GetConfigurationRequest,
	reply *sub.GetConfigurationResponse) error {
	return callGetConfiguration(client, request, reply)
}

func CallPoll(client *srpc.Client, request sub.PollRequest,
	reply *sub.PollResponse) error {
	return callPoll(client, request, reply)
}

func CallSetConfiguration(client *srpc.Client,
	request sub.SetConfigurationRequest,
	reply *sub.SetConfigurationResponse) error {
	return callSetConfiguration(client, request, reply)
}

func CallUpdate(client *srpc.Client, request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	return callUpdate(client, request, reply)
}
