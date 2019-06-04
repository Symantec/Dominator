package rpcd

import (
	"github.com/Symantec/Dominator/lib/image"
	"github.com/Symantec/Dominator/lib/srpc"
)

func (t *srpcType) ListDirectories(conn *srpc.Conn) error {
	for _, directory := range t.imageDataBase.ListDirectories() {
		if err := conn.Encode(directory); err != nil {
			return err
		}
	}
	return conn.Encode(image.Directory{})
}
