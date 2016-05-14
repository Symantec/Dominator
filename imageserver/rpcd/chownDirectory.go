package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) ChownDirectory(conn *srpc.Conn) error {
	var request imageserver.ChangeOwnerRequest
	var response imageserver.ChangeOwnerResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.changeOwner(request, &response, conn.Username()); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) changeOwner(request imageserver.ChangeOwnerRequest,
	reply *imageserver.ChangeOwnerResponse, username string) error {
	return t.imageDataBase.ChownDirectory(request.DirectoryName,
		request.OwnerGroup, username)
}
