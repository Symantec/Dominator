package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func (t *srpcType) CreateVm(conn *srpc.Conn) error {
	return t.manager.CreateVm(conn)
}
