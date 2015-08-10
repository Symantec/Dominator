package imageserver

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/common"
)

const (
	UNCOMPRESSED = iota
	GZIP
)

type DataStreamer interface {
	Size() uint64
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
}

type AddFilesRequest struct {
	Objects [][]byte
}

type AddFilesResponse struct {
	Hashes []hash.Hash
}

type AddImageRequest struct {
	ImageName       string
	Filter          []string
	CompressionType uint
	ImageData       DataStreamer
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
	Objects []hash.Hash
}

type GetFilesResponse struct {
	ObjectSizes []uint64
}

type GetImageRequest struct {
	ImageName string
}

type GetImageResponse struct {
	Image *image.Image
}

type ListImagesRequest struct {
}

type ListImagesResponse struct {
	ImageNames []string
}
