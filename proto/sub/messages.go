package sub

import (
	"github.com/Symantec/Dominator/sub/scanner"
)

type PollRequest struct {
	HaveGeneration uint64
}

type PollResponse struct {
	GenerationCount uint64
	FileSystem      *scanner.FileSystem
}
