package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) CreateVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	return t.manager.CreateVm(conn, decoder, encoder)
}
