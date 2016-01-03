package client

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func callPoll(client *srpc.Client, request sub.PollRequest,
	reply *sub.PollResponse) error {
	conn, err := client.Call("Subd.Poll")
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
	if err := gob.NewDecoder(conn).Decode(reply); err != nil {
		return err
	}
	if reply.FileSystemFollows {
		fs, err := filesystem.Decode(conn)
		if err != nil {
			return err
		}
		reply.FileSystem = fs
	}
	return nil
}
