package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
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
