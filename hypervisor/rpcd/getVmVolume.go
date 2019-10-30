package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
)

func (t *srpcType) GetVmVolume(conn *srpc.Conn) error {
	return t.manager.GetVmVolume(conn)
}
