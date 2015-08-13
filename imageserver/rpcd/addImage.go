package rpcd

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *rpcType) AddImage(request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	if imageDataBase.CheckImage(request.ImageName) {
		return errors.New("image already exists")
	}
	// Verify all objects are available.
	objectServer := imageDataBase.ObjectServer()
	for _, inode := range request.Image.FileSystem.RegularInodeTable {
		found, err := objectServer.CheckObject(inode.Hash)
		if err != nil {
			return err
		}
		if !found {
			return errors.New(fmt.Sprintf("object: %x is not available",
				inode.Hash))
		}
	}
	// TODO(rgooch): Remove debugging output.
	fmt.Printf("AddImage(%s)\n", request.ImageName)
	return imageDataBase.AddImage(request.Image, request.ImageName)
}
