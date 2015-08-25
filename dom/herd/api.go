package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/sub/scanner"
	"net/rpc"
)

type Sub struct {
	hostname                     string
	requiredImage                string
	plannedImage                 string
	connection                   *rpc.Client
	fileSystem                   *scanner.FileSystem
	generationCount              uint64
	generationCountAtChangeStart uint64
}

type Herd struct {
	imageServerAddress string
	nextSubToPoll      uint
	subsByName         map[string]*Sub
	subsByIndex        []*Sub
	imagesByName       map[string]*image.Image
}

func NewHerd(imageServerAddress string) *Herd {
	return &Herd{imageServerAddress: imageServerAddress}
}

func (herd *Herd) MdbUpdate(mdb *mdb.Mdb) {
	herd.mdbUpdate(mdb)
}

func (herd *Herd) PollNextSub() bool {
	return herd.pollNextSub()
}
