package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) GetRootCookiePath(conn *srpc.Conn,
	request hypervisor.GetRootCookiePathRequest,
	reply *hypervisor.GetRootCookiePathResponse) error {
	*reply = hypervisor.GetRootCookiePathResponse{
		Path: t.manager.GetRootCookiePath(),
	}
	return nil
}
