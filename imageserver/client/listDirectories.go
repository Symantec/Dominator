package client

import (
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func listDirectories(client *srpc.Client) ([]image.Directory, error) {
	conn, err := client.Call("ImageServer.ListDirectories")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	directories := make([]image.Directory, 0)
	for {
		var directory image.Directory
		if err := conn.Decode(&directory); err != nil {
			return nil, err
		}
		if directory.Name == "" {
			break
		}
		directories = append(directories, directory)
	}
	return directories, nil
}
