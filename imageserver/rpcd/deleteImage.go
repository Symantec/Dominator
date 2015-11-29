package rpcd

import (
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *rpcType) DeleteImage(request imageserver.DeleteImageRequest,
	reply *imageserver.DeleteImageResponse) error {
	var response imageserver.DeleteImageResponse
	err := t.imageDataBase.DeleteImage(request.ImageName)
	if err == nil {
		response.Success = true
	} else {
		response.Success = false
		response.ErrorString = err.Error()
	}
	*reply = response
	return nil
}
