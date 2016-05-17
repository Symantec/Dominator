package client

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func callChownDirectory(client *srpc.Client, dirname, ownerGroup string) error {
	conn, err := client.Call("ImageServer.ChownDirectory")
	if err != nil {
		return err
	}
	defer conn.Close()
	encoder := gob.NewEncoder(conn)
	request := imageserver.ChangeOwnerRequest{DirectoryName: dirname,
		OwnerGroup: ownerGroup}
	var reply imageserver.ChangeOwnerResponse
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
	if err := gob.NewDecoder(conn).Decode(&reply); err != nil {
		return err
	}
	return nil
}
