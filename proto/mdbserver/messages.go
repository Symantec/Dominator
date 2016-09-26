package mdbserver

import (
	"github.com/Symantec/Dominator/lib/mdb"
)

// The GetMdbUpdates() RPC is fully streamed.
// The client sends no information to the server.
// The server sends a stream of MdbUpdate messages.
// At connection start, the full MDB data are presented in .MachinesToAdd and
// .MachinesToUpdate and .MachinesToDelete will be nil.

type MdbUpdate struct {
	MachinesToAdd    []mdb.Machine
	MachinesToUpdate []mdb.Machine
	MachinesToDelete []string
}
