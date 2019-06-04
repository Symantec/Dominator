package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) ReplaceVmImage(conn *srpc.Conn) error {
	return t.manager.ReplaceVmImage(conn, conn.GetAuthInformation())
}
