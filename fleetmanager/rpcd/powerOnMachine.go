package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/fleetmanager"
)

func (t *srpcType) PowerOnMachine(conn *srpc.Conn,
	request fleetmanager.PowerOnMachineRequest,
	reply *fleetmanager.PowerOnMachineResponse) error {
	*reply = fleetmanager.PowerOnMachineResponse{
		errors.ErrorToString(t.hypervisorsManager.PowerOnMachine(
			request.Hostname, conn.GetAuthInformation()))}
	return nil
}
