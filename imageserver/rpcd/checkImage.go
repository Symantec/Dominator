package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) CheckImage(conn *srpc.Conn,
	request imageserver.CheckImageRequest,
	reply *imageserver.CheckImageResponse) error {
	var response imageserver.CheckImageResponse
	response.ImageExists = t.imageDataBase.CheckImage(request.ImageName)
	*reply = response
	return nil
}
