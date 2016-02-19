package sub

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectcache"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/common"
	"time"
)

type Configuration struct {
	ScanSpeedPercent    uint
	NetworkSpeedPercent uint
	ScanExclusionList   []string
}

type FetchRequest struct {
	ServerAddress string
	Hashes        []hash.Hash
}

type FetchResponse common.StatusResponse

type GetConfigurationRequest struct {
}

type GetConfigurationResponse Configuration

// The GetFiles() RPC is fully streamed.
// The client sends a stream of strings (filenames) it wants. An empty string
// signals the end of the stream.
// The server (the sub) sends a stream of GetFileResponse messages. No response
// is sent for the end-of-stream signal.

type GetFileResponse struct {
	Error error
	Size  uint64
}

type PollRequest struct {
	HaveGeneration uint64
	ShortPollOnly  bool // If true, do not send FileSystem or ObjectCache.
}

type PollResponse struct {
	NetworkSpeed                 uint64
	FetchInProgress              bool // Fetch() and Update() mutually exclusive
	UpdateInProgress             bool
	LastFetchError               string
	LastUpdateError              string
	LastUpdateHadTriggerFailures bool
	StartTime                    time.Time
	PollTime                     time.Time
	ScanCount                    uint64
	GenerationCount              uint64
	FileSystem                   *filesystem.FileSystem // Streamed separately.
	FileSystemFollows            bool
	ObjectCache                  objectcache.ObjectCache // Streamed separately.
} // FileSystem is encoded afterwards, followed by ObjectCache.

type SetConfigurationRequest Configuration

type SetConfigurationResponse common.StatusResponse

type FileToCopyToCache struct {
	Name string
	Hash hash.Hash
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
