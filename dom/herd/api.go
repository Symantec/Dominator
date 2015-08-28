package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/sub/scanner"
	"sync"
)

type Sub struct {
	herd                         *Herd
	hostname                     string
	requiredImage                string
	plannedImage                 string
	busyMutex                    sync.Mutex
	busy                         bool
	fileSystem                   *scanner.FileSystem
	generationCount              uint64
	generationCountAtChangeStart uint64
}

type Herd struct {
	imageServerAddress      string
	nextSubToPoll           uint
	subsByName              map[string]*Sub
	subsByIndex             []*Sub
	imagesByName            map[string]*image.Image
	makeConnectionSemaphore chan bool
	pollSemaphore           chan bool
}

func NewHerd(imageServerAddress string) *Herd {
	return newHerd(imageServerAddress)
}

func (herd *Herd) MdbUpdate(mdb *mdb.Mdb) {
	herd.mdbUpdate(mdb)
}

func (herd *Herd) PollNextSub() bool {
	return herd.pollNextSub()
}

func (herd *Herd) StartServer(portNum uint, daemon bool) error {
	return herd.startServer(portNum, daemon)
}
