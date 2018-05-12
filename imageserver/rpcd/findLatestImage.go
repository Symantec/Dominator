package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) FindLatestImage(conn *srpc.Conn,
	request imageserver.FindLatestImageRequest,
	reply *imageserver.FindLatestImageResponse) error {
	imageName, err := t.imageDataBase.FindLatestImage(request.DirectoryName,
		request.IgnoreExpiringImages)
	*reply = imageserver.FindLatestImageResponse{
		ImageName: imageName,
		Error:     errors.ErrorToString(err),
	}
	return nil
}
