package rpcd

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *rpcType) AddImage(request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	if imageDataBase.CheckImage(request.ImageName) {
		return errors.New("image already exists")
	}
	// Verify all objects are available.
	hashes := make([]hash.Hash, len(request.Image.FileSystem.RegularInodeTable))
	for index, inode := range request.Image.FileSystem.RegularInodeTable {
		hashes[index] = inode.Hash
	}
	objectsPresent, err := imageDataBase.ObjectServer().CheckObjects(hashes)
	if err != nil {
		return err
	}
	for index, present := range objectsPresent {
		if !present {
			return errors.New(fmt.Sprintf("object: %x is not available",
				hashes[index]))
		}
	}
	// TODO(rgooch): Remove debugging output.
	fmt.Printf("AddImage(%s)\n", request.ImageName)
	return imageDataBase.AddImage(request.Image, request.ImageName)
}
