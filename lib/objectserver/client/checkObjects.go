package client

import (
	"encoding/gob"
	"errors"
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
	conn, err := client.Call("ObjectServer.CheckObjects")
	if err != nil {
		return nil, err
	}
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		return nil, err
	}
	if err := conn.Flush(); err != nil {
		return nil, err
	}
	str, err := conn.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if str != "\n" {
		return nil, errors.New(str[:len(str)-1])
	}
	if err := gob.NewDecoder(conn).Decode(&reply); err != nil {
		return nil, err
	}
	return reply.ObjectSizes, nil
}
