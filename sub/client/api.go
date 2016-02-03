package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
)

func CallCleanup(client *srpc.Client, request sub.CleanupRequest,
	reply *sub.CleanupResponse) error {
	return callCleanup(client, request, reply)
}

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

func GetFiles(client *srpc.Client, filenames []string,
	readerFunc func(reader io.Reader, size uint64) error) error {
	return getFiles(client, filenames, readerFunc)
}
