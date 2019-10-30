package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) PrepareForCopy(conn *srpc.Conn,
	request proto.PrepareForCopyRequest,
	reply *proto.PrepareForCopyResponse) error {
	return t.unpacker.PrepareForCopy(request.StreamName)
}
