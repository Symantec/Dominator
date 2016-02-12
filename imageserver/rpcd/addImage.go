package rpcd

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) AddImage(conn *srpc.Conn) error {
	defer conn.Flush()
	var request imageserver.AddImageRequest
	var response imageserver.AddImageResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.addImage(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) addImage(request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	if t.replicationMaster != "" {
		return errors.New(replicationMessage + t.replicationMaster)
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
	t.logger.Printf("AddImage(%s)\n", request.ImageName)
	return t.imageDataBase.AddImage(request.Image, request.ImageName)
}
