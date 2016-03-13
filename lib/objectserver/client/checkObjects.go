package client

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/objectserver"
	"net/rpc"
)

func (objClient *ObjectClient) checkObjects(hashes []hash.Hash) (
	[]uint64, error) {
	var request objectserver.CheckObjectsRequest
	request.Hashes = hashes
	var reply objectserver.CheckObjectsResponse
	client, err := rpc.DialHTTP("tcp", objClient.address)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error dialing\t%s\n", err.Error()))
	}
	defer client.Close()
	err = client.Call("ObjectServer.CheckObjects", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.ObjectSizes, nil
}
