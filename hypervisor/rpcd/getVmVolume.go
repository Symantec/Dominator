package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) GetVmVolume(conn *srpc.Conn) error {
	return t.manager.GetVmVolume(conn)
}
