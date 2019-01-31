package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) CopyVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	if err := t.copyVm(conn, decoder, encoder); err != nil {
		return encoder.Encode(hypervisor.CopyVmResponse{Error: err.Error()})
	}
	return nil
}

func (t *srpcType) copyVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	var request hypervisor.CopyVmRequest
	if err := decoder.Decode(&request); err != nil {
		return err
	}
	return t.manager.CopyVm(conn, request, encoder)
}
