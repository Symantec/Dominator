package rpcd

import (
	"fmt"

	"github.com/Cloud-Foundations/Dominator/lib/constants"
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
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
