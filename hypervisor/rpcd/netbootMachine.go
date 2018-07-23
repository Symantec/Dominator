package rpcd

import (
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/errors"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

func (t *srpcType) NetbootMachine(conn *srpc.Conn,
	request hypervisor.NetbootMachineRequest,
	reply *hypervisor.NetbootMachineResponse) error {
	*reply = hypervisor.NetbootMachineResponse{
		errors.ErrorToString(t.netbootMachine(request))}
	return nil
}

func (t *srpcType) netbootMachine(
	request hypervisor.NetbootMachineRequest) error {
	err := t.dhcpServer.AddNetbootLease(request.Address, request.Hostname,
		request.Subnet)
	if err != nil {
		return err
	}
	t.tftpbootServer.RegisterFiles(request.Address.IpAddress, request.Files)
	if request.WaitTimeout <= 0 {
		request.WaitTimeout = time.Minute
	}
	if request.FilesExpiration < request.WaitTimeout+time.Second {
		request.FilesExpiration = request.WaitTimeout + time.Second
	}
	if len(request.Files) > 0 {
		go expireFilesAfter(t.tftpbootServer, request.Address.IpAddress,
			request.FilesExpiration)
	}
	if request.OfferExpiration < request.WaitTimeout+time.Second {
		request.OfferExpiration = request.WaitTimeout + time.Second
	}
	go expireLeaseAfter(t.dhcpServer, request.Address.IpAddress,
		request.OfferExpiration)
	timer := time.NewTimer(request.WaitTimeout)
	for count := 0; count < int(request.NumAcknowledgementsToWaitFor); {
		ackChannel := t.dhcpServer.MakeAcknowledgmentChannel(
			request.Address.IpAddress)
		select {
		case <-ackChannel:
			count++
		case <-timer.C:
			return errors.New("timed out receiving lease acknowledgement")
		}
	}
	return nil
}

func expireFilesAfter(tftpbootServer TftpbootServer, ipAddr net.IP,
	timeout time.Duration) {
	time.Sleep(timeout)
	tftpbootServer.UnregisterFiles(ipAddr)
}

func expireLeaseAfter(dhcpServer DhcpServer, ipAddr net.IP,
	timeout time.Duration) {
	time.Sleep(timeout)
	dhcpServer.RemoveLease(ipAddr)
}
