package imageserver

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/common"
)

type AddImageRequest struct {
	ImageName string
	Image     *image.Image
}

type AddImageResponse struct {
}

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
