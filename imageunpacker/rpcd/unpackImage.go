package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) UnpackImage(conn *srpc.Conn,
	request proto.UnpackImageRequest,
	reply *proto.UnpackImageResponse) error {
	return t.unpacker.UnpackImage(request.StreamName, request.ImageLeafName)
}
