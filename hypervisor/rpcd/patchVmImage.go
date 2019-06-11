package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
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
