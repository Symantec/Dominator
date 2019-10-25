package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func (t *srpcType) ChangeImageExpiration(conn *srpc.Conn,
	request imageserver.ChangeImageExpirationRequest,
	reply *imageserver.ChangeImageExpirationResponse) error {
	if err := t.checkMutability(); err != nil {
		reply.Error = errors.ErrorToString(err)
		return nil
	}
	_, err := t.imageDataBase.ChangeImageExpiration(
		request.ImageName, request.ExpiresAt, conn.GetAuthInformation())
	reply.Error = errors.ErrorToString(err)
	return nil
}

func (t *srpcType) GetImageExpiration(conn *srpc.Conn,
	request imageserver.GetImageExpirationRequest,
	reply *imageserver.GetImageExpirationResponse) error {
	if img := t.imageDataBase.GetImage(request.ImageName); img == nil {
		reply.Error = "image not found"
	} else {
		reply.ExpiresAt = img.ExpiresAt
	}
	return nil
}
