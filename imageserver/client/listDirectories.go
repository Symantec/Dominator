package client

import (
	"encoding/gob"

	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
)

func listDirectories(client *srpc.Client) ([]image.Directory, error) {
	conn, err := client.Call("ImageServer.ListDirectories")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	directories := make([]image.Directory, 0)
	decoder := gob.NewDecoder(conn)
	for {
		var directory image.Directory
		if err := decoder.Decode(&directory); err != nil {
			return nil, err
		}
		if directory.Name == "" {
			break
		}
		directories = append(directories, directory)
	}
	return directories, nil
}
