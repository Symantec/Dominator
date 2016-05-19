package rpcd

import (
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) AddImage(conn *srpc.Conn,
	request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	request.Image.CreatedBy = conn.Username() // Must always set this field.
	if err := t.checkMutability(); err != nil {
		return err
	}
	if t.imageDataBase.CheckImage(request.ImageName) {
		return errors.New("image already exists")
	}
	if request.Image == nil {
		return errors.New("nil image")
	}
	if request.Image.FileSystem == nil {
		return errors.New("nil file-system")
	}
	// Verify all objects are available.
	hashes := make([]hash.Hash, 0, request.Image.FileSystem.NumRegularInodes)
	for _, inode := range request.Image.FileSystem.InodeTable {
		if inode, ok := inode.(*filesystem.RegularInode); ok {
			if inode.Size > 0 {
				hashes = append(hashes, inode.Hash)
			}
		}
	}
	objectSizes, err := t.imageDataBase.ObjectServer().CheckObjects(hashes)
	if err != nil {
		return err
	}
	for index, size := range objectSizes {
		if size < 1 {
			return errors.New(fmt.Sprintf("object: %x is not available",
				hashes[index]))
		}
	}
	request.Image.FileSystem.RebuildInodePointers()
	username := request.Image.CreatedBy
	if username == "" {
		t.logger.Printf("AddImage(%s)\n", request.ImageName)
	} else {
		t.logger.Printf("AddImage(%s) by %s\n", request.ImageName, username)
	}
	return t.imageDataBase.AddImage(request.Image, request.ImageName, &username)
}
