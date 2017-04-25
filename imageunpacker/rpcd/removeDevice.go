package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func (t *srpcType) RemoveDevice(conn *srpc.Conn,
	request proto.RemoveDeviceRequest,
	reply *proto.RemoveDeviceResponse) error {
	return t.unpacker.RemoveDevice(request.DeviceId)
}
