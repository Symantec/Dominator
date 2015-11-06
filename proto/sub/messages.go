package sub

import (
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/triggers"
	"github.com/Symantec/Dominator/proto/common"
	"github.com/Symantec/Dominator/sub/scanner"
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

type PollRequest struct {
	HaveGeneration uint64
}

type PollResponse struct {
	NetworkSpeed     uint64
	FetchInProgress  bool // Fetch() and Update() are mutually exclusive.
	UpdateInProgress bool
	GenerationCount  uint64
	FileSystem       *scanner.FileSystem
}

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
	InodesToMake        []Inode
	HardlinksToMake     []Hardlink
	PathsToDelete       []string
	DirectoriesToMake   []Inode
	InodesToChange      []Inode
	MultiplyUsedObjects map[hash.Hash]uint64
	Triggers            *triggers.Triggers
}

type UpdateResponse struct{}

type CleanupRequest struct {
	Hashes []hash.Hash
}

type CleanupResponse struct{}
