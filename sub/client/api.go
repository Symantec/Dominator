package client

import (
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/sub"
)

func BoostCpuLimit(client *srpc.Client) error {
	return boostCpuLimit(client)
}

func Cleanup(client *srpc.Client, hashes []hash.Hash) error {
	return cleanup(client, hashes)
}

func Fetch(client *srpc.Client, serverAddress string,
	hashes []hash.Hash) error {
	return fetch(client, serverAddress, hashes)
}

func GetConfiguration(client *srpc.Client) (sub.Configuration, error) {
	return getConfiguration(client)
}

func CallPoll(client *srpc.Client, request sub.PollRequest,
	reply *sub.PollResponse) error {
	return callPoll(client, request, reply)
}

func SetConfiguration(client *srpc.Client, config sub.Configuration) error {
	return setConfiguration(client, config)
}

func CallUpdate(client *srpc.Client, request sub.UpdateRequest,
	reply *sub.UpdateResponse) error {
	return callUpdate(client, request, reply)
}

func GetFiles(client *srpc.Client, filenames []string,
	readerFunc func(reader io.Reader, size uint64) error) error {
	return getFiles(client, filenames, readerFunc)
}
