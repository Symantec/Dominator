package client

import (
	"encoding/gob"

	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func listUnreferencedObjects(client *srpc.Client) (
	map[hash.Hash]uint64, error) {
	conn, err := client.Call("ImageServer.ListUnreferencedObjects")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	objects := make(map[hash.Hash]uint64)
	decoder := gob.NewDecoder(conn)
	for {
		var object imageserver.Object
		if err := decoder.Decode(&object); err != nil {
			return nil, err
		}
		if object.Size < 1 {
			break
		}
		objects[object.Hash] = object.Size
	}
	return objects, nil
}
