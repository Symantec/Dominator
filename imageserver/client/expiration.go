package client

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func changeImageExpiration(client *srpc.Client, name string,
	expiresAt time.Time) error {
	request := imageserver.ChangeImageExpirationRequest{
		ImageName: name,
		ExpiresAt: expiresAt,
	}
	var reply imageserver.ChangeImageExpirationResponse
	err := client.RequestReply("ImageServer.ChangeImageExpiration", request,
		&reply)
	if err != nil {
		return err
	}
	return errors.New(reply.Error)
}

func getImageExpiration(client *srpc.Client, name string) (time.Time, error) {
	request := imageserver.GetImageExpirationRequest{ImageName: name}
	var reply imageserver.GetImageExpirationResponse
	err := client.RequestReply("ImageServer.GetImageExpiration", request,
		&reply)
	if err != nil {
		return time.Time{}, err
	}
	if err := errors.New(reply.Error); err != nil {
		return time.Time{}, err
	}
	return reply.ExpiresAt, nil
}
