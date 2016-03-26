/*
	Package mdbd implements a simple MDB watcher.

	Package mdbd may be used to read MDB data from a file and watch for updates.
*/
package mdbd

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
)

// StartMdbDaemon starts an in-process "daemon" goroutine which watches the
// file named by mdbFileName for MDB updates. The default format is JSON, but
// if the filename extension is ".gob" then GOB format is read.
// If the file is replaced by a different inode, MDB data are read from the new
// inode and if the MDB data are different than previously read, they are sent
// over the returned channel.
// The logger will be used to log problems.
func StartMdbDaemon(mdbFileName string, logger *log.Logger) <-chan *mdb.Mdb {
	return startMdbDaemon(mdbFileName, logger)
}
