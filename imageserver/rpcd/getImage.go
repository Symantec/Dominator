package rpcd

import (
	"time"

	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) GetImage(conn *srpc.Conn,
	request imageserver.GetImageRequest,
	reply *imageserver.GetImageResponse) error {
	var response imageserver.GetImageResponse
	response.Image = t.getImageNow(request)
	*reply = response
	if response.Image != nil || request.Timeout == 0 {
		return nil
	}
	// Image not found yet and willing to wait.
	addCh := t.imageDataBase.RegisterAddNotifier()
	defer func() {
		t.imageDataBase.UnregisterAddNotifier(addCh)
		select {
		case <-addCh:
		default:
		}
	}()
	timer := time.NewTimer(request.Timeout)
	for {
		select {
		case imageName := <-addCh:
			if imageName == request.ImageName {
				if !timer.Stop() {
					<-timer.C
				}
				response.Image = t.getImageNow(request)
				*reply = response
				return nil
			}
		case <-timer.C:
			return nil
		}
	}
}

func (t *srpcType) getImageNow(
	request imageserver.GetImageRequest) *image.Image {
	img := *t.imageDataBase.GetImage(request.ImageName)
	if request.IgnoreFilesystem {
		img.FileSystem = nil
	}
	return &img
}
