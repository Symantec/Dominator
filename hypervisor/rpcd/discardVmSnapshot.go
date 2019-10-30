package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) DiscardVmSnapshot(conn *srpc.Conn,
	request hypervisor.DiscardVmSnapshotRequest,
	reply *hypervisor.DiscardVmSnapshotResponse) error {
	response := hypervisor.DiscardVmSnapshotResponse{
		errors.ErrorToString(t.manager.DiscardVmSnapshot(request.IpAddress,
			conn.GetAuthInformation()))}
	*reply = response
	return nil
}
