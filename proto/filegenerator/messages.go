package filegenerator

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"time"
)

type YieldRequest struct {
	Machine   mdb.Machine
	Pathnames []string
}

type FileInfo struct {
	Pathname   string
	Hash       *hash.Hash // A nil pointer indicates missing or invalidation.
	ValidUntil time.Time
}
type YieldResponse struct {
	Hostname string
	Files    []FileInfo
}
