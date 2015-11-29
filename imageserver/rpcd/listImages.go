package rpcd

import (
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *rpcType) ListImages(request imageserver.ListImagesRequest,
	reply *imageserver.ListImagesResponse) error {
	var response imageserver.ListImagesResponse
	response.ImageNames = t.imageDataBase.ListImages()
	*reply = response
	return nil
}
