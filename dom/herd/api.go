package herd

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectcache"
	"io"
	"log"
	"sync"
	"time"
)

type subStatus uint

func (status subStatus) String() string {
	return status.string()
}

const (
	statusUnknown = iota
	statusConnecting
	statusDNSError
	statusConnectTimeout
	statusFailedToConnect
	statusWaitingToPoll
	statusPolling
	statusFailedToPoll
	statusSubNotReady
	statusImageNotReady
	statusFetching
	statusFailedToFetch
	statusComputingUpdate
	statusUpdating
	statusFailedToUpdate
	statusWaitingForNextFullPoll
	statusSynced
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

type Sub struct {
	herd                         *Herd
	hostname                     string
	requiredImage                string
	plannedImage                 string
	busyMutex                    sync.Mutex
	busy                         bool
	havePlannedImage             bool
	fileSystem                   *filesystem.FileSystem
	objectCache                  objectcache.ObjectCache
	generationCount              uint64
	generationCountAtChangeStart uint64
	status                       subStatus
	lastConnectionStartTime      time.Time
	lastConnectionSucceededTime  time.Time
	lastConnectDuration          time.Duration
	lastPollStartTime            time.Time
	lastPollSucceededTime        time.Time
	lastShortPollDuration        time.Duration
	lastFullPollDuration         time.Duration
	lastComputeUpdateCpuDuration time.Duration
}

type Herd struct {
	sync.RWMutex         // Protect map and slice mutations.
	imageServerAddress   string
	logger               *log.Logger
	htmlWriters          []HtmlWriter
	nextSubToPoll        uint
	subsByName           map[string]*Sub
	subsByIndex          []*Sub // Sorted by Sub.hostname.
	imagesByName         map[string]*image.Image
	missingImages        map[string]time.Time
	connectionSemaphore  chan bool
	pollSemaphore        chan bool
	currentScanStartTime time.Time
	previousScanDuration time.Duration
}

func NewHerd(imageServerAddress string, logger *log.Logger) *Herd {
	return newHerd(imageServerAddress, logger)
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

func (herd *Herd) AddHtmlWriter(htmlWriter HtmlWriter) {
	herd.addHtmlWriter(htmlWriter)
}
