package rpcd

import (
	"fmt"

	"github.com/Symantec/Dominator/lib/constants"
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/fleetmanager"
)

func (t *srpcType) GetHypervisorForVM(conn *srpc.Conn,
	request proto.GetHypervisorForVMRequest,
	reply *proto.GetHypervisorForVMResponse) error {
	hypervisor, err := t.hypervisorsManager.GetHypervisorForVm(
		request.IpAddress)
	response := proto.GetHypervisorForVMResponse{
		Error: errors.ErrorToString(err),
	}
	if err == nil {
		response.HypervisorAddress = fmt.Sprintf("%s:%d",
			hypervisor, constants.HypervisorPortNumber)
	}
	*reply = response
	return nil
}
