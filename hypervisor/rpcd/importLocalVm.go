package rpcd

import (
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ImportLocalVm(conn *srpc.Conn,
	request hypervisor.ImportLocalVmRequest,
	reply *hypervisor.ImportLocalVmResponse) error {
	*reply = hypervisor.ImportLocalVmResponse{
		errors.ErrorToString(t.importLocalVm(conn, request))}
	return nil
}

func (t *srpcType) importLocalVm(conn *srpc.Conn,
	request hypervisor.ImportLocalVmRequest) error {
	if err := testIfLoopback(conn); err != nil {
		return err
	}
	return t.manager.ImportLocalVm(conn.GetAuthInformation(), request)
}

func testIfLoopback(conn *srpc.Conn) error {
	host, _, err := net.SplitHostPort(conn.RemoteAddr())
	if err != nil {
		return err
	}
	if !net.ParseIP(host).IsLoopback() {
		return errors.New("local connection required")
	}
	return nil
}
