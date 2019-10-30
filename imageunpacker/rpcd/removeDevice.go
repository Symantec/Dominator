package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) RemoveDevice(conn *srpc.Conn,
	request proto.RemoveDeviceRequest,
	reply *proto.RemoveDeviceResponse) error {
	return t.unpacker.RemoveDevice(request.DeviceId)
}
