package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/imageunpacker"
)

func (t *srpcType) PrepareForUnpack(conn *srpc.Conn,
	request proto.PrepareForUnpackRequest,
	reply *proto.PrepareForUnpackResponse) error {
	return t.unpacker.PrepareForUnpack(request.StreamName,
		request.SkipIfPrepared, request.DoNotWaitForResult)
}
