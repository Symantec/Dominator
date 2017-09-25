package client

import (
	"encoding/gob"
	"io"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/sub"
)

func getFiles(client *srpc.Client, filenames []string,
	readerFunc func(reader io.Reader, size uint64) error) error {
	conn, err := client.Call("Subd.GetFiles")
	if err != nil {
		return err
	}
	defer conn.Close()
	go sendRequests(conn, filenames)
	decoder := gob.NewDecoder(conn)
	for range filenames {
		var reply sub.GetFileResponse
		if err := decoder.Decode(&reply); err != nil {
			return err
		}
		if reply.Error != nil {
			return reply.Error
		}
		if err := readerFunc(&io.LimitedReader{R: conn, N: int64(reply.Size)},
			reply.Size); err != nil {
			return err
		}
	}
	return nil
}

func sendRequests(conn *srpc.Conn, filenames []string) error {
	for _, filename := range filenames {
		if _, err := conn.WriteString(filename + "\n"); err != nil {
			return err
		}
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return conn.Flush()
}
