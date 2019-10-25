package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) GetStatus(conn *srpc.Conn, request proto.GetStatusRequest,
	reply *proto.GetStatusResponse) error {
	*reply = t.unpacker.GetStatus()
	return nil
}
