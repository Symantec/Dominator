package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) MigrateVm(conn *srpc.Conn) error {
	if err := t.manager.MigrateVm(conn); err != nil {
		return conn.Encode(hypervisor.MigrateVmResponse{Error: err.Error()})
	}
	return conn.Encode(hypervisor.MigrateVmResponse{Final: true})
}
