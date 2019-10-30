package sub

import (
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/filesystem"
	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/objectcache"
	"github.com/Cloud-Foundations/Dominator/lib/triggers"
)

type BoostCpuLimitRequest struct{}

type BoostCpuLimitResponse struct{}

type Configuration struct {
	CpuPercent          uint
	NetworkSpeedPercent uint
	ScanSpeedPercent    uint
	ScanExclusionList   []string
}

type FetchRequest struct {
	ServerAddress string
	Wait          bool
	Hashes        []hash.Hash
}

type FetchResponse struct{}

type GetConfigurationRequest struct{}

type GetConfigurationResponse Configuration

// The GetFiles() RPC is fully streamed.
// The client sends a stream of strings (filenames) it wants. An empty string
// signals the end of the stream.
// The server (the sub) sends a stream of GetFileResponse messages. No response
// is sent for the end-of-stream signal.

type GetFileResponse struct {
	Error error
	Size  uint64
} // File data are streamed afterwards.

type PollRequest struct {
	HaveGeneration uint64
	ShortPollOnly  bool // If true, do not send FileSystem or ObjectCache.
}

type PollResponse struct {
	NetworkSpeed                 uint64 // Capacity of the network interface.
	CurrentConfiguration         Configuration
	FetchInProgress              bool // Fetch() and Update() mutually exclusive
	UpdateInProgress             bool
	LastFetchError               string
	LastUpdateError              string
	LastUpdateHadTriggerFailures bool
	LastSuccessfulImageName      string
	FreeSpace                    *uint64
	StartTime                    time.Time
	PollTime                     time.Time
	ScanCount                    uint64
	DurationOfLastScan           time.Duration
	GenerationCount              uint64
	FileSystemFollows            bool
	FileSystem                   *filesystem.FileSystem  // Streamed separately.
	ObjectCache                  objectcache.ObjectCache // Streamed separately.
} // FileSystem is encoded afterwards, followed by ObjectCache.

type SetConfigurationRequest Configuration

type SetConfigurationResponse struct{}

type FileToCopyToCache struct {
	Name       string
	Hash       hash.Hash
	DoHardlink bool
}

type Hardlink struct {
	NewLink string
	Target  string
}

type Inode struct {
	Name string
	filesystem.GenericInode
}

type UpdateRequest struct {
	ImageName string
	Wait      bool
	// The ordering here reflects the ordering that the sub is expected to use.
	FilesToCopyToCache  []FileToCopyToCache
	DirectoriesToMake   []Inode
	InodesToMake        []Inode
	HardlinksToMake     []Hardlink
	PathsToDelete       []string
	InodesToChange      []Inode
	MultiplyUsedObjects map[hash.Hash]uint64
	Triggers            *triggers.Triggers
}

type UpdateResponse struct{}

type CleanupRequest struct {
	Hashes []hash.Hash
}

type CleanupResponse struct{}
