package rpcd

import (
	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	proto "github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ScanVmRoot(conn *srpc.Conn,
	request proto.ScanVmRootRequest,
	reply *proto.ScanVmRootResponse) error {
	fs, err := t.manager.ScanVmRoot(request.IpAddress,
		conn.GetAuthInformation(), request.Filter)
	*reply = proto.ScanVmRootResponse{
		Error:      errors.ErrorToString(err),
		FileSystem: fs,
	}
	return nil
}
