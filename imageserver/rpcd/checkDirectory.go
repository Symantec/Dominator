package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func (t *srpcType) CheckDirectory(conn *srpc.Conn,
	request imageserver.CheckDirectoryRequest,
	reply *imageserver.CheckDirectoryResponse) error {
	response := imageserver.CheckDirectoryResponse{
		t.imageDataBase.CheckDirectory(request.DirectoryName)}
	*reply = response
	return nil
}
