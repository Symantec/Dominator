package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func (t *srpcType) UnpackImage(conn *srpc.Conn,
	request proto.UnpackImageRequest,
	reply *proto.UnpackImageResponse) error {
	return t.unpacker.UnpackImage(request.StreamName, request.ImageLeafName)
}
