package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) ReplaceVmImage(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	return t.manager.ReplaceVmImage(conn, decoder, encoder,
		conn.GetAuthInformation())
}
