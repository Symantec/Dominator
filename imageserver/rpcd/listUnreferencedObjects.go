package rpcd

import (
	"encoding/gob"

	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/imageserver"
)

func (t *srpcType) ListUnreferencedObjects(conn *srpc.Conn) error {
	encoder := gob.NewEncoder(conn)
	for hashVal, size := range t.imageDataBase.ListUnreferencedObjects() {
		obj := imageserver.Object{Hash: hashVal, Size: size}
		if err := encoder.Encode(obj); err != nil {
			return err
		}
	}
	return encoder.Encode(imageserver.Object{})
}
