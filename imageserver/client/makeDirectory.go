package client

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func callMakeDirectory(client *srpc.Client, dirname string) error {
	conn, err := client.Call("ImageServer.MakeDirectory")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	request := imageserver.MakeDirectoryRequest{dirname}
	var reply imageserver.MakeDirectoryResponse
	if err := encoder.Encode(request); err != nil {
		return err
	}
	conn.Flush()
	str, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	if str != "\n" {
		return errors.New(str[:len(str)-1])
	}
	return gob.NewDecoder(conn).Decode(&reply)
}
