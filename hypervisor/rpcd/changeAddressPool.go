package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeAddressPool(conn *srpc.Conn,
	request hypervisor.ChangeAddressPoolRequest,
	reply *hypervisor.ChangeAddressPoolResponse) error {
	*reply = hypervisor.ChangeAddressPoolResponse{
		Error: errors.ErrorToString(t.changeAddressPool(conn, request))}
	return nil
}

func (t *srpcType) changeAddressPool(conn *srpc.Conn,
	request hypervisor.ChangeAddressPoolRequest) error {
	if len(request.AddressesToAdd) > 0 {
		err := t.manager.AddAddressesToPool(request.AddressesToAdd)
		if err != nil {
			return err
		}
	}
	if len(request.AddressesToRemove) > 0 {
		err := t.manager.RemoveAddressesFromPool(request.AddressesToRemove)
		if err != nil {
			return err
		}
	}
	if len(request.MaximumFreeAddresses) > 0 {
		err := t.manager.RemoveExcessAddressesFromPool(
			request.MaximumFreeAddresses)
		if err != nil {
			return err
		}
	}
	return nil
}
