package client

import (
	"fmt"
	"io"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imaginator"
)

func buildImage(client *srpc.Client, request proto.BuildImageRequest,
	response *proto.BuildImageResponse, logWriter io.Writer) error {
	conn, err := client.Call("Imaginator.BuildImage")
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := conn.Encode(request); err != nil {
		return err
	}
	if err := conn.Flush(); err != nil {
		return err
	}
	str, err := conn.ReadString('\n')
	if err != nil {
		return err
	}
	if str != "\n" {
		return errors.New(str[:len(str)-1])
	}
	for {
		var reply proto.BuildImageResponse
		if err := conn.Decode(&reply); err != nil {
			return fmt.Errorf("error reading reply: %s", err)
		}
		if err := errors.New(reply.ErrorString); err != nil {
			return err
		}
		logWriter.Write(reply.BuildLog)
		reply.BuildLog = nil
		if reply.Image != nil || reply.ImageName != "" {
			*response = reply
			return nil
		}
	}
}
