package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) GetImageUpdates(conn *srpc.Conn) error {
	defer conn.Flush()
	encoder := gob.NewEncoder(conn)
	for _, imageName := range t.imageDataBase.ListImages() {
		var imageUpdate imageserver.ImageUpdate
		imageUpdate.Name = imageName
		if err := encoder.Encode(imageUpdate); err != nil {
			return err
		}
	}
	conn.Flush()
	addChannel := t.imageDataBase.RegisterAddNotifier()
	deleteChannel := t.imageDataBase.RegisterDeleteNotifier()
	defer t.imageDataBase.UnregisterAddNotifier(addChannel)
	defer t.imageDataBase.UnregisterDeleteNotifier(deleteChannel)
	for {
		select {
		case imageName := <-addChannel:
			if err := sendUpdate(encoder, imageName,
				imageserver.OperationAddImage); err != nil {
				return err
			}
		case imageName := <-deleteChannel:
			if err := sendUpdate(encoder, imageName,
				imageserver.OperationDeleteImage); err != nil {
				return err
			}
		}
		conn.Flush()
	}
}

func sendUpdate(encoder *gob.Encoder, name string, operation uint) error {
	var imageUpdate imageserver.ImageUpdate
	imageUpdate.Name = name
	imageUpdate.Operation = operation
	return encoder.Encode(imageUpdate)
}
