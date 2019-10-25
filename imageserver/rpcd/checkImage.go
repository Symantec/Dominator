package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func (t *srpcType) CheckImage(conn *srpc.Conn,
	request imageserver.CheckImageRequest,
	reply *imageserver.CheckImageResponse) error {
	var response imageserver.CheckImageResponse
	response.ImageExists = t.imageDataBase.CheckImage(request.ImageName)
	*reply = response
	return nil
}
