package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/objectserver"
)

func (t *srpcType) CheckObjects(conn *srpc.Conn,
	request objectserver.CheckObjectsRequest,
	reply *objectserver.CheckObjectsResponse) error {
	sizes, err := t.objectServer.CheckObjects(request.Hashes)
	if err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	reply.ObjectSizes = sizes
	return nil
}
