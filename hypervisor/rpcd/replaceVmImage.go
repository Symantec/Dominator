package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func (t *srpcType) ReplaceVmImage(conn *srpc.Conn) error {
	return t.manager.ReplaceVmImage(conn, conn.GetAuthInformation())
}
