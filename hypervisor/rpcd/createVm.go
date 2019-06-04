package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) CreateVm(conn *srpc.Conn) error {
	return t.manager.CreateVm(conn)
}
