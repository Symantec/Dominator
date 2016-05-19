package client

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func listImages(client *srpc.Client) ([]string, error) {
	conn, err := client.Call("ImageServer.ListImages")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	images := make([]string, 0)
	for {
		line, err := conn.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = line[:len(line)-1]
		if line == "" {
			break
		}
		images = append(images, line)
	}
	return images, nil
}
