package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/objectserver/rpcd/lib"
	"runtime"
)

func (t *srpcType) AddObjects(conn *srpc.Conn) error {
	defer runtime.GC() // An opportune time to take out the garbage.
	return lib.AddObjects(conn, t.objectServer, t.logger)
}
