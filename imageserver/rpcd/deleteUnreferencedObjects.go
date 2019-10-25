package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/format"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/imageserver"
)

func (t *srpcType) DeleteUnreferencedObjects(conn *srpc.Conn,
	request imageserver.DeleteUnreferencedObjectsRequest,
	reply *imageserver.DeleteUnreferencedObjectsResponse) error {
	username := conn.Username()
	if username == "" {
		t.logger.Printf("DeleteUnreferencedObjects(%d%%, %s)\n",
			request.Percentage, format.FormatBytes(request.Bytes))
	} else {
		t.logger.Printf("DeleteUnreferencedObjects(%d%%, %s) by %s\n",
			request.Percentage, format.FormatBytes(request.Bytes), username)
	}
	return t.imageDataBase.DeleteUnreferencedObjects(request.Percentage,
		request.Bytes)
}
