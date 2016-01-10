package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) GetImage(conn *srpc.Conn) error {
	defer conn.Flush()
	var request imageserver.GetImageRequest
	var response imageserver.GetImageResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if err := t.getImage(request, &response); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	return gob.NewEncoder(conn).Encode(response)
}

func (t *srpcType) getImage(request imageserver.GetImageRequest,
	reply *imageserver.GetImageResponse) error {
	var response imageserver.GetImageResponse
	response.Image = t.imageDataBase.GetImage(request.ImageName)
	*reply = response
	return nil
}
