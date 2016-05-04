package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) CheckImage(conn *srpc.Conn) error {
	var request imageserver.CheckImageRequest
	var response imageserver.CheckImageResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.checkImage(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) checkImage(request imageserver.CheckImageRequest,
	reply *imageserver.CheckImageResponse) error {
	var response imageserver.CheckImageResponse
	response.ImageExists = t.imageDataBase.CheckImage(request.ImageName)
	*reply = response
	return nil
}
