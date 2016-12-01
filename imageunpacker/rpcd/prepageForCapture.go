package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func (t *srpcType) PrepareForCapture(conn *srpc.Conn,
	request proto.PrepareForCaptureRequest,
	reply *proto.PrepareForCaptureResponse) error {
	return t.unpacker.PrepareForCapture(request.StreamName)
}
