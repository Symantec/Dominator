package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
)

func (t *srpcType) GetImageUpdates(conn *srpc.Conn) error {
	defer conn.Flush()
	t.logger.Println("New image replication client connected")
	t.incrementNumReplicationClients(true)
	defer t.incrementNumReplicationClients(false)
	addChannel := t.imageDataBase.RegisterAddNotifier()
	deleteChannel := t.imageDataBase.RegisterDeleteNotifier()
	mkdirChannel := t.imageDataBase.RegisterMakeDirectoryNotifier()
	defer t.imageDataBase.UnregisterAddNotifier(addChannel)
	defer t.imageDataBase.UnregisterDeleteNotifier(deleteChannel)
	defer t.imageDataBase.UnregisterMakeDirectoryNotifier(mkdirChannel)
	encoder := gob.NewEncoder(conn)
	directories := t.imageDataBase.ListDirectories()
	image.SortDirectories(directories)
	for _, directory := range directories {
		imageUpdate := imageserver.ImageUpdate{
			Directory: &directory,
			Operation: imageserver.OperationMakeDirectory,
		}
		if err := encoder.Encode(imageUpdate); err != nil {
			t.logger.Println(err)
			return err
		}
	}
	for _, imageName := range t.imageDataBase.ListImages() {
		imageUpdate := imageserver.ImageUpdate{Name: imageName}
		if err := encoder.Encode(imageUpdate); err != nil {
			t.logger.Println(err)
			return err
		}
	}
	// Signal end of initial image list.
	if err := encoder.Encode(imageserver.ImageUpdate{}); err != nil {
		t.logger.Println(err)
		return err
	}
	if err := conn.Flush(); err != nil {
		t.logger.Println(err)
		return err
	}
	t.logger.Println(
		"Finished sending initial image list to replication client")
	closeChannel := getCloseNotifier(conn)
	for {
		select {
		case imageName := <-addChannel:
			if err := sendUpdate(encoder, imageName,
				imageserver.OperationAddImage); err != nil {
				t.logger.Println(err)
				return err
			}
		case imageName := <-deleteChannel:
			if err := sendUpdate(encoder, imageName,
				imageserver.OperationDeleteImage); err != nil {
				t.logger.Println(err)
				return err
			}
		case directory := <-mkdirChannel:
			if err := sendMakeDirectory(encoder, directory); err != nil {
				t.logger.Println(err)
				return err
			}
		case err := <-closeChannel:
			if err == io.EOF {
				t.logger.Println("Image replication client disconnected")
				return nil
			}
			t.logger.Println(err)
			return err
		}
		if err := conn.Flush(); err != nil {
			t.logger.Println(err)
			return err
		}
	}
}

func (t *srpcType) incrementNumReplicationClients(increment bool) {
	t.numReplicationClientsLock.Lock()
	defer t.numReplicationClientsLock.Unlock()
	if increment {
		t.numReplicationClients++
	} else {
		t.numReplicationClients--
	}
}

func getCloseNotifier(conn *srpc.Conn) <-chan error {
	closeChannel := make(chan error)
	go func() {
		for {
			buf := make([]byte, 1)
			if _, err := conn.Read(buf); err != nil {
				closeChannel <- err
				return
			}
		}
	}()
	return closeChannel
}

func sendUpdate(encoder *gob.Encoder, name string, operation uint) error {
	imageUpdate := imageserver.ImageUpdate{Name: name, Operation: operation}
	return encoder.Encode(imageUpdate)
}

func sendMakeDirectory(encoder *gob.Encoder, directory image.Directory) error {
	imageUpdate := imageserver.ImageUpdate{
		Directory: &directory,
		Operation: imageserver.OperationMakeDirectory,
	}
	return encoder.Encode(imageUpdate)
}
