package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
)

type Sub struct {
	hostname        string
	requiredImage   string
	plannedImage    string
	connection      *rpc.Client
	fileSystem      *scanner.FileSystem
	generationCount uint64
}

type Herd struct {
	ImageServerAddress string
	nextSubToPoll      uint
	subsByName         map[string]*Sub
	subsByIndex        []*Sub
	imageByName        map[string]*image.Image
}

func (herd *Herd) MdbUpdate(mdb *mdb.Mdb) {
	herd.mdbUpdate(mdb)
}

func (herd *Herd) PollNextSub() bool {
	return herd.pollNextSub()
}
