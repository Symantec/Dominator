package rpcd

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"github.com/proxypoke/group.go"
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
	if username == "" {
		return errors.New("no username: unauthenticated connection")
	}
	if _, err := group.Lookup(request.OwnerGroup); err != nil {
		return err
	}
	t.logger.Printf("ChownDirectory(%s) to: %s by %s\n",
		request.DirectoryName, request.OwnerGroup, username)
	return t.imageDataBase.ChownDirectory(request.DirectoryName,
		request.OwnerGroup)
}
