package client

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func callAddObjects(client *srpc.Client, request objectserver.AddObjectsRequest,
	reply *objectserver.AddObjectsResponse) error {
	conn, err := client.Call("ObjectServer.AddObjects")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	if err := encoder.Encode(request); err != nil {
		return err
	}
	conn.Flush()
	str, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	if str != "\n" {
		return errors.New(str)
	}
	return gob.NewDecoder(conn).Decode(reply)
}
