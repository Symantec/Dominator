package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ChangeVmTags(conn *srpc.Conn,
	request hypervisor.ChangeVmTagsRequest,
	reply *hypervisor.ChangeVmTagsResponse) error {
	response := hypervisor.ChangeVmTagsResponse{
		errors.ErrorToString(
			t.manager.ChangeVmTags(request.IpAddress, conn.GetAuthInformation(),
				request.Tags))}
	*reply = response
	return nil
}
