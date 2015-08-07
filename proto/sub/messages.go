package sub

import (
	"github.com/Symantec/Dominator/proto/common"
	"github.com/Symantec/Dominator/sub/scanner"
)

type Configuration struct {
	ScanSpeedPercent  uint
	ScanExclusionList []string
}

type FetchRequest struct {
	ServerHostname   string
	ServerPortNumber uint
	Objects          []common.Hash
}

type FetchResponse common.StatusResponse

type GetConfigurationRequest struct {
}

type GetConfigurationResponse Configuration

type PollRequest struct {
	HaveGeneration uint64
}

type PollResponse struct {
	GenerationCount uint64
	FileSystem      *scanner.FileSystem
}

type SetConfigurationRequest Configuration

type SetConfigurationResponse common.StatusResponse
