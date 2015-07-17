package fleet

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

type Fleet struct {
	nextSubToPoll uint
	subs          []*Sub
}

func (fleet *Fleet) MdbUpdate(mdb *mdb.Mdb) {
	fleet.mdbUpdate(mdb)
}

func (fleet *Fleet) PollNextSub() {
	fleet.pollNextSub()
}
