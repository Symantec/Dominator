package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) ExportImage(conn *srpc.Conn,
	request proto.ExportImageRequest,
	reply *proto.ExportImageResponse) error {
	return t.unpacker.ExportImage(request.StreamName, request.Type,
		request.Destination)
}
