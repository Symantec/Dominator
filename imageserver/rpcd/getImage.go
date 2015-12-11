package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) GetImage(conn *srpc.Conn) {
	defer conn.Flush()
	var request imageserver.GetImageRequest
	var response imageserver.GetImageResponse
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	if err := t.getImage(request, &response); err != nil {
		conn.WriteString(err.Error() + "\n")
		return
	}
	conn.WriteString("\n")
	encoder := gob.NewEncoder(conn)
	encoder.Encode(response)
}

func (t *srpcType) getImage(request imageserver.GetImageRequest,
	reply *imageserver.GetImageResponse) error {
	var response imageserver.GetImageResponse
	response.Image = t.imageDataBase.GetImage(request.ImageName)
	*reply = response
	return nil
}
