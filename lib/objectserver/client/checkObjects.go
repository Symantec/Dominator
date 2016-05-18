package client

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (objClient *ObjectClient) checkObjects(hashes []hash.Hash) (
	[]uint64, error) {
	var request objectserver.CheckObjectsRequest
	request.Hashes = hashes
	var reply objectserver.CheckObjectsResponse
	client, err := srpc.DialHTTP("tcp", objClient.address, 0)
	if err != nil {
		return nil, fmt.Errorf("error dialing: %s\n", err)
	}
	defer client.Close()
	err = client.RequestReply("ObjectServer.CheckObjects", request, &reply)
	if err != nil {
		return nil, err
	}
	return reply.ObjectSizes, nil
}
