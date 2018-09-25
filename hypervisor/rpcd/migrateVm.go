package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) MigrateVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	if err := t.manager.MigrateVm(conn, decoder, encoder); err != nil {
		return encoder.Encode(hypervisor.MigrateVmResponse{Error: err.Error()})
	}
	return encoder.Encode(hypervisor.MigrateVmResponse{Final: true})
}
