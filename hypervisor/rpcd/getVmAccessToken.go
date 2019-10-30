package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) GetVmAccessToken(conn *srpc.Conn,
	request hypervisor.GetVmAccessTokenRequest,
	reply *hypervisor.GetVmAccessTokenResponse) error {
	token, err := t.manager.GetVmAccessToken(request.IpAddress,
		conn.GetAuthInformation(), request.Lifetime)
	response := hypervisor.GetVmAccessTokenResponse{
		Token: token,
		Error: errors.ErrorToString(err),
	}
	*reply = response
	return nil
}
