package rpcd

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) GetImageUpdates(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	defer conn.Flush()
	t.logger.Printf("New image replication client connected from: %s\n",
		conn.RemoteAddr())
	t.incrementNumReplicationClients(true)
	defer t.incrementNumReplicationClients(false)
	addChannel := t.imageDataBase.RegisterAddNotifier()
	deleteChannel := t.imageDataBase.RegisterDeleteNotifier()
	mkdirChannel := t.imageDataBase.RegisterMakeDirectoryNotifier()
	defer t.imageDataBase.UnregisterAddNotifier(addChannel)
	defer t.imageDataBase.UnregisterDeleteNotifier(deleteChannel)
	defer t.imageDataBase.UnregisterMakeDirectoryNotifier(mkdirChannel)
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
	closeChannel := conn.GetCloseNotifier()
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
			if err == nil {
				t.logger.Printf("Image replication client disconnected: %s\n",
					conn.RemoteAddr())
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

func sendUpdate(encoder srpc.Encoder, name string, operation uint) error {
	imageUpdate := imageserver.ImageUpdate{Name: name, Operation: operation}
	return encoder.Encode(imageUpdate)
}

func sendMakeDirectory(encoder srpc.Encoder, directory image.Directory) error {
	imageUpdate := imageserver.ImageUpdate{
		Directory: &directory,
		Operation: imageserver.OperationMakeDirectory,
	}
	return encoder.Encode(imageUpdate)
}
