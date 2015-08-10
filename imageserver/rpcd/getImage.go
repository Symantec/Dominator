package rpcd

import (
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *ImageServer) GetImage(request imageserver.GetImageRequest,
	reply *imageserver.GetImageResponse) error {
	var response imageserver.GetImageResponse
	response.Image = imageDataBase.GetImage(request.ImageName)
	*reply = response
	return nil
}
