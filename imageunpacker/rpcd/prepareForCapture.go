package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) PrepareForCapture(conn *srpc.Conn,
	request proto.PrepareForCaptureRequest,
	reply *proto.PrepareForCaptureResponse) error {
	return t.unpacker.PrepareForCapture(request.StreamName)
}
