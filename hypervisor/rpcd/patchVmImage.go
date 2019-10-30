package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) PatchVmImage(conn *srpc.Conn) error {
	var request hypervisor.PatchVmImageRequest
	if err := conn.Decode(&request); err != nil {
		return err
	}
	return conn.Encode(hypervisor.PatchVmImageResponse{
		Error: errors.ErrorToString(t.manager.PatchVmImage(conn, request)),
		Final: true,
	})
	return nil
}
