package client

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
	"io"
)

func Cleanup(client *srpc.Client, hashes []hash.Hash) error {
	return cleanup(client, hashes)
}

func Fetch(client *srpc.Client, serverAddress string,
	hashes []hash.Hash) error {
	return fetch(client, serverAddress, hashes)
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
