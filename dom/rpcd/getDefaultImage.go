package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/dominator"
)

func (t *rpcType) GetDefaultImage(conn *srpc.Conn,
	request dominator.GetDefaultImageRequest,
	reply *dominator.GetDefaultImageResponse) error {
	reply.ImageName = t.herd.GetDefaultImage()
	return nil
}
