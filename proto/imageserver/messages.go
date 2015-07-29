package imageserver

import (
	"github.com/Symantec/Dominator/proto/common"
)

const (
	UNCOMPRESSED = iota
	GZIP
)

type AddRequest struct {
	ImageName       string
	Filter          [][]string
	DataSize        uint64
	CompressionType uint
}

type AddResponse common.StatusResponse

type CheckRequest struct {
	ImageName string
}

type CheckResponse struct {
	ImageExists bool
}

type DeleteRequest struct {
	ImageName string
}

type DeleteResponse common.StatusResponse

type ListRequest struct {
}

type ListResponse struct {
	ImageNames []string
}
