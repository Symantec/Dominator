package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func (t *srpcType) MakeDirectory(conn *srpc.Conn,
	request imageserver.MakeDirectoryRequest,
	reply *imageserver.MakeDirectoryResponse) error {
	username := conn.Username()
	if err := t.checkMutability(); err != nil {
		return err
	}
	if username == "" {
		t.logger.Printf("MakeDirectory(%s)\n", request.DirectoryName)
	} else {
		t.logger.Printf("MakeDirectory(%s) by %s\n",
			request.DirectoryName, username)
	}
	return t.imageDataBase.MakeDirectory(request.DirectoryName,
		conn.GetAuthInformation())
}
