package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
	"net/rpc"
)

type Sub struct {
	hostname        string
	requiredImage   string
	plannedImage    string
	connection      *rpc.Client
	generationCount uint64
}

type Herd struct {
	nextSubToPoll uint
	subsByName    map[string]*Sub
	subsByIndex   []*Sub
}

func (herd *Herd) MdbUpdate(mdb *mdb.Mdb) {
	herd.mdbUpdate(mdb)
}

func (herd *Herd) PollNextSub() {
	herd.pollNextSub()
}
