package rpcd

import (
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"github.com/proxypoke/group.go"
)

func (t *srpcType) ChownDirectory(conn *srpc.Conn,
	request imageserver.ChangeOwnerRequest,
	reply *imageserver.ChangeOwnerResponse) error {
	username := conn.Username()
	if username == "" {
		return errors.New("no username: unauthenticated connection")
	}
	if request.OwnerGroup != "" {
		if _, err := group.Lookup(request.OwnerGroup); err != nil {
			return err
		}
	}
	t.logger.Printf("ChownDirectory(%s) to: \"%s\" by %s\n",
		request.DirectoryName, request.OwnerGroup, username)
	return t.imageDataBase.ChownDirectory(request.DirectoryName,
		request.OwnerGroup)
}
