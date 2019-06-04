package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) ListUnreferencedObjects(conn *srpc.Conn) error {
	for hashVal, size := range t.imageDataBase.ListUnreferencedObjects() {
		obj := imageserver.Object{Hash: hashVal, Size: size}
		if err := conn.Encode(obj); err != nil {
			return err
		}
	}
	return conn.Encode(imageserver.Object{})
}
