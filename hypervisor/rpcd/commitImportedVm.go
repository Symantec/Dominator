package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) CommitImportedVm(conn *srpc.Conn,
	request hypervisor.CommitImportedVmRequest,
	reply *hypervisor.CommitImportedVmResponse) error {
	*reply = hypervisor.CommitImportedVmResponse{
		errors.ErrorToString(
			t.manager.CommitImportedVm(request.IpAddress,
				conn.GetAuthInformation()))}
	return nil
}
