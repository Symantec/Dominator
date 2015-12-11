package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) DeleteImage(conn *srpc.Conn) {
	defer conn.Flush()
	var request imageserver.DeleteImageRequest
	var response imageserver.DeleteImageResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	if err := t.deleteImage(request, &response); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	conn.WriteString("\n")
	encoder := gob.NewEncoder(conn)
	encoder.Encode(response)
}

func (t *srpcType) deleteImage(request imageserver.DeleteImageRequest,
	reply *imageserver.DeleteImageResponse) error {
	var response imageserver.DeleteImageResponse
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
