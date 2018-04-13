package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) AddAddressesToPool(conn *srpc.Conn,
	request hypervisor.AddAddressesToPoolRequest,
	reply *hypervisor.AddAddressesToPoolResponse) error {
	response := hypervisor.AddAddressesToPoolResponse{
		errors.ErrorToString(t.manager.AddAddressesToPool(request.Addresses))}
	*reply = response
	return nil
}
