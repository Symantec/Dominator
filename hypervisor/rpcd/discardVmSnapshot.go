package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
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
