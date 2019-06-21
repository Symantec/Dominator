package rpcd

import (
	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) ExportLocalVm(conn *srpc.Conn,
	request hypervisor.ExportLocalVmRequest,
	reply *hypervisor.ExportLocalVmResponse) error {
	vmInfo, err := t.exportLocalVm(conn, request)
	*reply = hypervisor.ExportLocalVmResponse{
		Error:  errors.ErrorToString(err),
		VmInfo: *vmInfo,
	}
	return nil
}

func (t *srpcType) exportLocalVm(conn *srpc.Conn,
	request hypervisor.ExportLocalVmRequest) (
	*hypervisor.ExportLocalVmInfo, error) {
	if err := testIfLoopback(conn); err != nil {
		return nil, err
	}
	return t.manager.ExportLocalVm(conn.GetAuthInformation(), request)
}
