package herd

import (
	"github.com/Symantec/Dominator/dom/images"
	filegenclient "github.com/Symantec/Dominator/lib/filegen/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	filegenproto "github.com/Symantec/Dominator/proto/filegenerator"
	subproto "github.com/Symantec/Dominator/proto/sub"
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
	statusConnectionRefused
	statusNoRouteToHost
	statusConnectTimeout
	statusMissingCertificate
	statusBadCertificate
	statusFailedToConnect
	statusWaitingToPoll
	statusPolling
	statusPollDenied
	statusFailedToPoll
	statusSubNotReady
	statusImageUndefined
	statusImageNotReady
	statusFetching
	statusFetchDenied
	statusFailedToFetch
	statusPushing
	statusPushDenied
	statusFailedToPush
	statusFailedToGetObject
	statusComputingUpdate
	statusSendingUpdate
	statusMissingComputedFile
	statusUpdatesDisabled
	statusUnsafeUpdate
	statusUpdating
	statusUpdateDenied
	statusFailedToUpdate
	statusWaitingForNextFullPoll
	statusSynced
)

type HtmlWriter interface {
	WriteHtml(writer io.Writer)
}

type Sub struct {
	herd                         *Herd
	mdb                          mdb.Machine
	requiredImageName            string // Updated only by sub goroutine.
	plannedImageName             string // Updated only by sub goroutine.
	clientResource               *srpc.ClientResource
	computedInodes               map[string]*filesystem.RegularInode
	fileUpdateChannel            <-chan []filegenproto.FileInfo
	busyFlagMutex                sync.Mutex
	busy                         bool
	busyMutex                    sync.Mutex
	deleting                     bool
	busyStartTime                time.Time
	busyStopTime                 time.Time
	havePlannedImage             bool
	startTime                    time.Time
	pollTime                     time.Time
	fileSystem                   *filesystem.FileSystem
	objectCache                  objectcache.ObjectCache
	generationCount              uint64
	computedFilesChangeTime      time.Time
	scanCountAtLastUpdateEnd     uint64
	isInsecure                   bool
	status                       subStatus
	publishedStatus              subStatus
	pendingSafetyClear           bool
	lastConnectionStartTime      time.Time
	lastReachableTime            time.Time
	lastConnectionSucceededTime  time.Time
	lastConnectDuration          time.Duration
	lastPollStartTime            time.Time
	lastPollSucceededTime        time.Time
	lastShortPollDuration        time.Duration
	lastFullPollDuration         time.Duration
	lastPollWasFull              bool
	lastScanDuration             time.Duration
	lastComputeUpdateCpuDuration time.Duration
	lastUpdateTime               time.Time
	lastSyncTime                 time.Time
	lastSuccessfulImageName      string
}

func (sub *Sub) String() string {
	return sub.string()
}

type Herd struct {
	sync.RWMutex          // Protect map and slice mutations.
	imageManager          *images.Manager
	objectServer          objectserver.ObjectServer
	computedFilesManager  *filegenclient.Manager
	logger                *log.Logger
	htmlWriters           []HtmlWriter
	updatesDisabledReason string
	updatesDisabledBy     string
	updatesDisabledTime   time.Time
	defaultImageName      string
	nextDefaultImageName  string
	configurationForSubs  subproto.Configuration
	nextSubToPoll         uint
	subsByName            map[string]*Sub
	subsByIndex           []*Sub // Sorted by Sub.hostname.
	pollSemaphore         chan struct{}
	pushSemaphore         chan struct{}
	computeSemaphore      chan struct{}
	currentScanStartTime  time.Time
	previousScanDuration  time.Duration
}

func NewHerd(imageServerAddress string, objectServer objectserver.ObjectServer,
	logger *log.Logger) *Herd {
	return newHerd(imageServerAddress, objectServer, logger)
}

func (herd *Herd) AddHtmlWriter(htmlWriter HtmlWriter) {
	herd.addHtmlWriter(htmlWriter)
}

func (herd *Herd) ClearSafetyShutoff(hostname string) error {
	return herd.clearSafetyShutoff(hostname)
}

func (herd *Herd) ConfigureSubs(configuration subproto.Configuration) error {
	return herd.configureSubs(configuration)
}

func (herd *Herd) DisableUpdates(username, reason string) error {
	return herd.disableUpdates(username, reason)
}

func (herd *Herd) EnableUpdates() error {
	return herd.enableUpdates()
}

func (herd *Herd) GetDefaultImage() string {
	return herd.defaultImageName
}

func (herd *Herd) GetSubsConfiguration() subproto.Configuration {
	return herd.getSubsConfiguration()
}

func (herd *Herd) MdbUpdate(mdb *mdb.Mdb) {
	herd.mdbUpdate(mdb)
}

func (herd *Herd) PollNextSub() bool {
	return herd.pollNextSub()
}

func (herd *Herd) SetDefaultImage(imageName string) error {
	return herd.setDefaultImage(imageName)
}

func (herd *Herd) StartServer(portNum uint, daemon bool) error {
	return herd.startServer(portNum, daemon)
}
