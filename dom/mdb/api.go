package mdb

import (
	"io"
	"log"
)

type Machine struct {
	Hostname      string
	RequiredImage string
	PlannedImage  string
}

type Mdb struct {
	Machines []Machine
}

func (mdb *Mdb) Len() int {
	return len(mdb.Machines)
}

func (mdb *Mdb) DebugWrite(w io.Writer) error {
	return mdb.debugWrite(w)
}

func StartMdbDaemon(mdbDir string, logger *log.Logger) <-chan *Mdb {
	return startMdbDaemon(mdbDir, logger)
}
