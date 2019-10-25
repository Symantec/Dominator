package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) CopyVm(conn *srpc.Conn) error {
	if err := t.copyVm(conn); err != nil {
		return conn.Encode(hypervisor.CopyVmResponse{Error: err.Error()})
	}
	return nil
}

func (t *srpcType) copyVm(conn *srpc.Conn) error {
	var request hypervisor.CopyVmRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	return t.manager.CopyVm(conn, request)
}
