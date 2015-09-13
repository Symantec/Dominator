package sub

import (
	"github.com/Symantec/Dominator/lib/hash"
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

type Directory struct {
	Name string
	Mode uint32
	Uid  uint32
	Gid  uint32
}

type UpdateRequest struct {
	PathsToDelete     []string
	DirectoriesToMake []Directory
}

type UpdateResponse struct{}
