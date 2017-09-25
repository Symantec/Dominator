/*
	Package mdbd implements a simple MDB watcher.

	Package mdbd may be used to read MDB data from a file or remote server and
	watch for updates.
*/
package mdbd

import (
	"flag"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
)

var (
	mdbServerHostname = flag.String("mdbServerHostname", "",
		"Hostname of remote MDB server to get MDB updates from")
	mdbServerPortNum = flag.Uint("mdbServerPortNum",
		constants.SimpleMdbServerPortNumber, "Port number of MDB server")
)

// StartMdbDaemon starts an in-process "daemon" goroutine which watches for MDB
// updates. At startup it will read the file named by mdbFileName for MDB data.
// The default format is JSON, but if the filename extension is ".gob" then GOB
// format is read. If the file is present and contains MDB data, the MDB data
// are sent over the returned channel, otherwise no MDB data are sent initially.
//
// By default the file is monitored for updates and if the file is replaced by a
// different inode, MDB data are read from the new inode. If the MDB data are
// different than previously read, they are sent over the channel. This mode of
// operation is designed for consuming MDB data via the file-system from a local
// mdbd daemon.
//
// Alternatively, if a remote MDB server is specified with the
// -mdbServerHostname and -mdbServerPortNum command-line flags then the remote
// MDB server is queried for MDB data. As MDB updates are received they are
// saved in the file and sent over the channel. In this mode of operation the
// file is read only once at startup. The file acts as a local cache of the MDB
// data received from the server, in case the MDB server is not available at
// a subsequent restart of the application.
//
// The logger will be used to log problems.
func StartMdbDaemon(mdbFileName string, logger log.Logger) <-chan *mdb.Mdb {
	return startMdbDaemon(mdbFileName, logger)
}
