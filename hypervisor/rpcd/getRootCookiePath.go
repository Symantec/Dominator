package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) GetRootCookiePath(conn *srpc.Conn,
	request hypervisor.GetRootCookiePathRequest,
	reply *hypervisor.GetRootCookiePathResponse) error {
	*reply = hypervisor.GetRootCookiePathResponse{
		Path: t.manager.GetRootCookiePath(),
	}
	return nil
}
