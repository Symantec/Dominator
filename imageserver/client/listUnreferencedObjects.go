package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func listUnreferencedObjects(client *srpc.Client) (
	map[hash.Hash]uint64, error) {
	conn, err := client.Call("ImageServer.ListUnreferencedObjects")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	objects := make(map[hash.Hash]uint64)
	for {
		var object imageserver.Object
		if err := conn.Decode(&object); err != nil {
			return nil, err
		}
		if object.Size < 1 {
			break
		}
		objects[object.Hash] = object.Size
	}
	return objects, nil
}
