package rpcd

import (
	"runtime"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/objectserver/rpcd/lib"
)

func (t *srpcType) AddObjects(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	defer runtime.GC() // An opportune time to take out the garbage.
	if t.replicationMaster == "" {
		return lib.AddObjects(conn, decoder, encoder, t.objectServer, t.logger)
	}
	return lib.AddObjectsWithMaster(conn, decoder, encoder, t.objectServer,
		t.replicationMaster, t.logger)
}
