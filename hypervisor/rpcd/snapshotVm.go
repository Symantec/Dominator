package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) SnapshotVm(conn *srpc.Conn,
	request hypervisor.SnapshotVmRequest,
	reply *hypervisor.SnapshotVmResponse) error {
	err := t.manager.SnapshotVm(request.IpAddress, conn.GetAuthInformation(),
		request.ForceIfNotStopped, request.RootOnly)
	*reply = hypervisor.SnapshotVmResponse{errors.ErrorToString(err)}
	return nil
}
