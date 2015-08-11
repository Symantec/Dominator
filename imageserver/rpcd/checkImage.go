package rpcd

import (
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *rpcType) CheckImage(request imageserver.CheckImageRequest,
	reply *imageserver.CheckImageResponse) error {
	var response imageserver.CheckImageResponse
	response.ImageExists = imageDataBase.CheckImage(request.ImageName)
	*reply = response
	return nil
}
