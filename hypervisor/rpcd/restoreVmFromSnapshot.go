package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) RestoreVmFromSnapshot(conn *srpc.Conn,
	request hypervisor.RestoreVmFromSnapshotRequest,
	reply *hypervisor.RestoreVmFromSnapshotResponse) error {
	response := hypervisor.RestoreVmFromSnapshotResponse{
		errors.ErrorToString(t.manager.RestoreVmFromSnapshot(request.IpAddress,
			conn.GetAuthInformation(), request.ForceIfNotStopped))}
	*reply = response
	return nil
}
