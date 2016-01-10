package mdbd

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
)

func StartMdbDaemon(mdbFileName string, logger *log.Logger) <-chan *mdb.Mdb {
	return startMdbDaemon(mdbFileName, logger)
}
