package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) RemoveExcessAddressesFromPool(conn *srpc.Conn,
	request hypervisor.RemoveExcessAddressesFromPoolRequest,
	reply *hypervisor.RemoveExcessAddressesFromPoolResponse) error {
	response := hypervisor.RemoveExcessAddressesFromPoolResponse{
		errors.ErrorToString(t.manager.RemoveExcessAddressesFromPool(
			request.MaximumFreeAddresses))}
	*reply = response
	return nil
}
