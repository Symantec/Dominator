package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
	"io"
)

func (t *srpcType) GetImageUpdates(conn *srpc.Conn) error {
	defer conn.Flush()
	t.logger.Println("New replication client connected")
	encoder := gob.NewEncoder(conn)
	for _, imageName := range t.imageDataBase.ListImages() {
		var imageUpdate imageserver.ImageUpdate
		imageUpdate.Name = imageName
		if err := encoder.Encode(imageUpdate); err != nil {
			t.logger.Println(err)
			return err
		}
	}
	// Signal end of initial image list.
	var imageUpdate imageserver.ImageUpdate
	if err := encoder.Encode(imageUpdate); err != nil {
		t.logger.Println(err)
		return err
	}
	if err := conn.Flush(); err != nil {
		t.logger.Println(err)
		return err
	}
	t.logger.Println(
		"Finished sending initial image list to replication client")
	addChannel := t.imageDataBase.RegisterAddNotifier()
	deleteChannel := t.imageDataBase.RegisterDeleteNotifier()
	defer t.imageDataBase.UnregisterAddNotifier(addChannel)
	defer t.imageDataBase.UnregisterDeleteNotifier(deleteChannel)
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
		case err := <-closeChannel:
			if err == io.EOF {
				t.logger.Println("Replication client disconnected")
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
	var imageUpdate imageserver.ImageUpdate
	imageUpdate.Name = name
	imageUpdate.Operation = operation
	return encoder.Encode(imageUpdate)
}
