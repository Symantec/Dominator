package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) PrepareVmForMigration(conn *srpc.Conn,
	request hypervisor.PrepareVmForMigrationRequest,
	reply *hypervisor.PrepareVmForMigrationResponse) error {
	*reply = hypervisor.PrepareVmForMigrationResponse{
		Error: errors.ErrorToString(
			t.manager.PrepareVmForMigration(request.IpAddress,
				conn.GetAuthInformation(), request.AccessToken,
				request.Enable)),
	}
	return nil
}
