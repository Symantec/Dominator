package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/dominator"
)

func (t *rpcType) GetDefaultImage(conn *srpc.Conn,
	request dominator.GetDefaultImageRequest,
	reply *dominator.GetDefaultImageResponse) error {
	reply.ImageName = t.herd.GetDefaultImage()
	return nil
}
