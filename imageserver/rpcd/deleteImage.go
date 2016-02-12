package rpcd

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) DeleteImage(conn *srpc.Conn) error {
	defer conn.Flush()
	var request imageserver.DeleteImageRequest
	var response imageserver.DeleteImageResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.deleteImage(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) deleteImage(request imageserver.DeleteImageRequest,
	reply *imageserver.DeleteImageResponse) error {
	var response imageserver.DeleteImageResponse
	if t.replicationMaster != "" {
		return errors.New(replicationMessage + t.replicationMaster)
	}
	err := t.imageDataBase.DeleteImage(request.ImageName)
	if err == nil {
		response.Success = true
	} else {
		response.Success = false
		response.ErrorString = err.Error()
	}
	*reply = response
	return nil
}
