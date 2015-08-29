package herd

import (
	"github.com/Symantec/Dominator/dom/mdb"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/sub/scanner"
	"sync"
	"time"
)

const (
	statusUnknown = iota
	statusConnecting
	statusFailedToConnect
	statusWaitingToPoll
	statusPolling
	statusFailedToPoll
	statusSubNotReady
	statusImageNotReady
	statusFetching
	statusFailedToFetch
	statusWaitingForNextPoll
	statusUpdating
	statusFailedToUpdate
	statusSynced
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
	status                       uint
	lastSuccessfulPoll           time.Time
}

type Herd struct {
	sync.RWMutex            // Protect map and slice mutations.
	imageServerAddress      string
	nextSubToPoll           uint
	subsByName              map[string]*Sub
	subsByIndex             []*Sub
	imagesByName            map[string]*image.Image
	makeConnectionSemaphore chan bool
	pollSemaphore           chan bool
	previousScanStartTime   time.Time
	currentScanStartTime    time.Time
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
