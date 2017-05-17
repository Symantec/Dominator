package rpcd

import (
	"errors"
	"fmt"
	iclient "github.com/Symantec/Dominator/imageserver/client"
	"github.com/Symantec/Dominator/lib/filesystem"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"time"
)

func (t *srpcType) AddImage(conn *srpc.Conn,
	request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
	request.Image.CreatedBy = conn.Username() // Must always set this field.
	request.Image.CreatedOn = time.Now()      // Must always set this field.
	return t.AddImageTrusted(conn, request, reply)
}

func (t *srpcType) AddImageTrusted(conn *srpc.Conn,
	request imageserver.AddImageRequest,
	reply *imageserver.AddImageResponse) error {
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
	t.setImageInjectionState(request.ImageName, true)
	defer t.setImageInjectionState(request.ImageName, false)
	if err := t.injectImage(conn, request); err != nil {
		return err
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

func (t *srpcType) injectImage(conn *srpc.Conn,
	request imageserver.AddImageRequest) error {
	if t.replicationMaster == "" {
		return nil
	}
	masterClient, err := srpc.DialHTTP("tcp", t.replicationMaster, 0)
	if err != nil {
		return err
	}
	return iclient.AddImageTrusted(masterClient, request.ImageName,
		request.Image)
}

func (t *srpcType) setImageInjectionState(name string, injecting bool) {
	t.imagesBeingInjectedLock.Lock()
	defer t.imagesBeingInjectedLock.Unlock()
	if injecting {
		t.imagesBeingInjected[name] = struct{}{}
	} else {
		defer delete(t.imagesBeingInjected, name)
	}
}
