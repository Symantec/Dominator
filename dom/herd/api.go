package herd

import (
	"io"
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/dom/images"
	"github.com/Cloud-Foundations/Dominator/lib/cpusharer"
	filegenclient "github.com/Cloud-Foundations/Dominator/lib/filegen/client"
	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/image"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/mdb"
	"github.com/Cloud-Foundations/Dominator/lib/net"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	filegenproto "github.com/Cloud-Foundations/Dominator/proto/filegenerator"
	subproto "github.com/Cloud-Foundations/Dominator/proto/sub"
	"github.com/Cloud-Foundations/tricorder/go/tricorder"
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
	statusNotEnoughFreeSpace
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
	requiredImageName            string       // Updated only by sub goroutine.
	requiredImage                *image.Image // Updated only by sub goroutine.
	plannedImageName             string       // Updated only by sub goroutine.
	plannedImage                 *image.Image // Updated only by sub goroutine.
	clientResource               *srpc.ClientResource
	computedInodes               map[string]*filesystem.RegularInode
	fileUpdateChannel            <-chan []filegenproto.FileInfo
	busyFlagMutex                sync.Mutex
	busy                         bool
	deletingFlagMutex            sync.Mutex
	deleting                     bool
	busyStartTime                time.Time
	busyStopTime                 time.Time
	cancelChannel                chan struct{}
	havePlannedImage             bool
	startTime                    time.Time
	pollTime                     time.Time
	fileSystem                   *filesystem.FileSystem
	objectCache                  objectcache.ObjectCache
	generationCount              uint64
	freeSpaceThreshold           *uint64
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
	logger                log.DebugLogger
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
	cpuSharer             *cpusharer.FifoCpuSharer
	dialer                net.Dialer
	currentScanStartTime  time.Time
	previousScanDuration  time.Duration
}

func NewHerd(imageServerAddress string, objectServer objectserver.ObjectServer,
	metricsDir *tricorder.DirectorySpec, logger log.DebugLogger) *Herd {
	return newHerd(imageServerAddress, objectServer, metricsDir, logger)
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

func (herd *Herd) LockWithTimeout(timeout time.Duration) {
	herd.lockWithTimeout(timeout)
}

func (herd *Herd) MdbUpdate(mdb *mdb.Mdb) {
	herd.mdbUpdate(mdb)
}

func (herd *Herd) PollNextSub() bool {
	return herd.pollNextSub()
}

func (herd *Herd) RLockWithTimeout(timeout time.Duration) {
	herd.rLockWithTimeout(timeout)
}

func (herd *Herd) SetDefaultImage(imageName string) error {
	return herd.setDefaultImage(imageName)
}

func (herd *Herd) StartServer(portNum uint, daemon bool) error {
	return herd.startServer(portNum, daemon)
}
