package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) ListImages(conn *srpc.Conn) error {
	for _, name := range t.imageDataBase.ListImages() {
		if _, err := conn.WriteString(name + "\n"); err != nil {
			return err
		}
	}
	_, err := conn.WriteString("\n")
	return err
}

func (t *rpcType) ListImages(request imageserver.ListImagesRequest,
	reply *imageserver.ListImagesResponse) error {
	var response imageserver.ListImagesResponse
	response.ImageNames = t.imageDataBase.ListImages()
	*reply = response
	return nil
}
