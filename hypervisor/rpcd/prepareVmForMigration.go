package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
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
