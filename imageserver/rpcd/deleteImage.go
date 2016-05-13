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
	if err := t.deleteImage(request, &response, conn.Username()); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) deleteImage(request imageserver.DeleteImageRequest,
	reply *imageserver.DeleteImageResponse, username string) error {
	var response imageserver.DeleteImageResponse
	if err := t.checkMutability(); err != nil {
		return err
	}
	if !t.imageDataBase.CheckImage(request.ImageName) {
		return errors.New("image does not exist")
	}
	if username == "" {
		t.logger.Printf("DeleteImage(%s)\n", request.ImageName)
	} else {
		t.logger.Printf("DeleteImage(%s) by %s\n", request.ImageName, username)
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
