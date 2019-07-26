package rpcd

import (
	"errors"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) DeleteImage(conn *srpc.Conn,
	request imageserver.DeleteImageRequest,
	reply *imageserver.DeleteImageResponse) error {
	username := conn.Username()
	if err := t.checkMutability(); err != nil {
		return err
	}
	if !t.imageDataBase.CheckImage(request.ImageName) {
		return errors.New("image does not exist")
	}
	if username == "" {
		t.logger.Printf("DeleteImage(%s)\n", request.ImageName)
	} else {
		t.logger.Printf("DeleteImage(%s) by %s\n", request.ImageName, username)
	}
	return t.imageDataBase.DeleteImage(request.ImageName,
		conn.GetAuthInformation())
}
