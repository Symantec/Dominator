package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) DiscardVmAccessToken(conn *srpc.Conn,
	request hypervisor.DiscardVmAccessTokenRequest,
	reply *hypervisor.DiscardVmAccessTokenResponse) error {
	*reply = hypervisor.DiscardVmAccessTokenResponse{
		errors.ErrorToString(t.manager.DiscardVmAccessToken(request.IpAddress,
			conn.GetAuthInformation(), request.AccessToken))}
	return nil
}
