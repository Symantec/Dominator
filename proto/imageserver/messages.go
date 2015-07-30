package imageserver

import (
	"github.com/Symantec/Dominator/proto/common"
)

const (
	UNCOMPRESSED = iota
	GZIP
)

type AddImageRequest struct {
	ImageName       string
	Filter          [][]string
	DataSize        uint64
	CompressionType uint
}

type AddImageResponse common.StatusResponse

type CheckImageRequest struct {
	ImageName string
}

type CheckImageResponse struct {
	ImageExists bool
}

type DeleteImageRequest struct {
	ImageName string
}

type DeleteImageResponse common.StatusResponse

type GetFilesRequest struct {
	Objects []common.Hash
}

type GetFilesResponse struct {
	ObjectSizes []uint64
}

type GetImageRequest struct {
	ImageName string
}

type ListImagesRequest struct {
}

type ListImagesResponse struct {
	ImageNames []string
}
