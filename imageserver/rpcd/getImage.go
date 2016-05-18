package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) GetImage(conn *srpc.Conn,
	request imageserver.GetImageRequest,
	reply *imageserver.GetImageResponse) error {
	var response imageserver.GetImageResponse
	response.Image = t.imageDataBase.GetImage(request.ImageName)
	*reply = response
	return nil
}
