package client

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func callCheckImage(client *srpc.Client, name string) (bool, error) {
	conn, err := client.Call("ImageServer.CheckImage")
	if err != nil {
		return false, err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	request := imageserver.CheckImageRequest{name}
	var reply imageserver.CheckImageResponse
	if err := encoder.Encode(request); err != nil {
		return false, err
	}
	conn.Flush()
	str, err := conn.ReadString('\n')
	if err != nil {
		return false, err
	}
	if str != "\n" {
		return false, errors.New(str[:len(str)-1])
	}
	if err := gob.NewDecoder(conn).Decode(&reply); err != nil {
		return false, err
	}
	return reply.ImageExists, nil
}
