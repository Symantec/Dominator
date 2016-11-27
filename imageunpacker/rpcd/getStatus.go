package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/imageunpacker"
)

func (t *srpcType) GetStatus(conn *srpc.Conn, request proto.GetStatusRequest,
	reply *proto.GetStatusResponse) error {
	*reply = t.unpacker.GetStatus()
	return nil
}
