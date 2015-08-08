package rpcd

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *ImageServer) AddImage(request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	var response imageserver.AddImageResponse
	if imageDataBase.CheckImage(request.ImageName) {
		return errors.New("image already exists")
	}
	if request.ImageData == nil {
		return errors.New("image data missing")
	}
	// TODO(rgooch): Implement the streamer support.
	fmt.Printf("AddImage(): Size=%d\n", request.ImageData.Size()) // HACK
	response.Success = true
	*reply = response
	return nil
}
