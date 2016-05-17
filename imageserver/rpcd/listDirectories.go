package rpcd

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) ListDirectories(conn *srpc.Conn) error {
	encoder := gob.NewEncoder(conn)
	for _, directory := range t.imageDataBase.ListDirectories() {
		if err := encoder.Encode(directory); err != nil {
			return err
		}
	}
	return encoder.Encode(image.Directory{})
}
