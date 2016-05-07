package imageserver

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/proto/common"
)

type AddImageRequest struct {
	ImageName string
	Image     *image.Image
}

type AddImageResponse struct{}

type ChangeOwnerRequest struct {
	DirectoryName string
	OwnerGroup    string
}

type ChangeOwnerResponse struct{}

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

const (
	OperationAddImage = iota
	OperationDeleteImage
	OperationMakeDirectory
)

// The GetImageUpdates() RPC is fully streamed.
// The client sends no information to the server.
// The server sends a stream of ImageUpdate messages.

type ImageUpdate struct {
	Name      string // "" signifies initial list is sent, changes to follow.
	Operation uint
}

type MakeDirectoryRequest struct {
	DirectoryName string
}

type MakeDirectoryResponse struct{}

// The ListImages() RPC is fully streamed.
// The client sends no information to the server.
// The server sends a stream of strings (image names) with an empty string
// signifying the end of the list.
