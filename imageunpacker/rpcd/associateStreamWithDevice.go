package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func (t *srpcType) AssociateStreamWithDevice(conn *srpc.Conn,
	request proto.AssociateStreamWithDeviceRequest,
	reply *proto.AssociateStreamWithDeviceResponse) error {
	return t.unpacker.AssociateStreamWithDevice(request.StreamName,
		request.DeviceId)
}
