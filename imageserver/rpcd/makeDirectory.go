package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) MakeDirectory(conn *srpc.Conn) error {
	defer conn.Flush()
	var request imageserver.MakeDirectoryRequest
	var response imageserver.MakeDirectoryResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.makeDirectory(request, &response,
		conn.GetUsername()); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) makeDirectory(request imageserver.MakeDirectoryRequest,
	reply *imageserver.MakeDirectoryResponse, username string) error {
	if err := t.checkMutability(); err != nil {
		return err
	}
	if username == "" {
		t.logger.Printf("MakeDirectory(%s)\n", request.DirectoryName)
	} else {
		t.logger.Printf("MakeDirectory(%s) by %s\n",
			request.DirectoryName, username)
	}
	return t.imageDataBase.MakeDirectory(request.DirectoryName, username, true)
}
