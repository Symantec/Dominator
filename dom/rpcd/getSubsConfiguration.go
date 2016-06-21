package rpcd

import (
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/dominator"
)

func (t *rpcType) GetSubsConfiguration(conn *srpc.Conn,
	request dominator.GetSubsConfigurationRequest,
	reply *dominator.GetSubsConfigurationResponse) error {
	*reply = dominator.GetSubsConfigurationResponse(
		t.herd.GetSubsConfiguration())
	return nil
}
