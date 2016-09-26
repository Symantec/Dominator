/*
	Package mdbd implements a simple MDB watcher.

	Package mdbd may be used to read MDB data from a file or remote server and
	watch for updates.
*/
package mdbd

import (
	"flag"
	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
)

var (
	mdbServerHostname = flag.String("mdbServerHostname", "",
		"Hostname of remote MDB server to get MDB updates from")
	mdbServerPortNum = flag.Uint("mdbServerPortNum",
		constants.SimpleMdbServerPortNumber, "Port number of MDB server")
)

// StartMdbDaemon starts an in-process "daemon" goroutine which watches the
// file named by mdbFileName for MDB updates. The default format is JSON, but
// if the filename extension is ".gob" then GOB format is read.
// If the file is replaced by a different inode, MDB data are read from the new
// inode and if the MDB data are different than previously read, they are sent
// over the returned channel.
// If a remote MDB server is specified with the -mdbServerHostname and
// -mdbServerPortNum command-line flags then the file named by mdbFileName is
// not watched for MDB updates and is only read once at process startup and
// thereafter the remote MDB server is queried for MDB data. As MDB updates are
// received they are saved in the file and sent over the returned channel.
// The logger will be used to log problems.
func StartMdbDaemon(mdbFileName string, logger *log.Logger) <-chan *mdb.Mdb {
	return startMdbDaemon(mdbFileName, logger)
}
