package rpcd

import (
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) ListVMs(conn *srpc.Conn,
	request hypervisor.ListVMsRequest,
	reply *hypervisor.ListVMsResponse) error {
	ipAddressStrings := t.manager.ListVMs(request.OwnerUsers, request.Sort)
	ipAddresses := make([]net.IP, 0, len(ipAddressStrings))
	for _, ipAddressString := range ipAddressStrings {
		ipAddress := net.ParseIP(ipAddressString)
		if shrunkIP := ipAddress.To4(); shrunkIP != nil {
			ipAddress = shrunkIP
		}
		ipAddresses = append(ipAddresses, ipAddress)
	}
	*reply = hypervisor.ListVMsResponse{IpAddresses: ipAddresses}
	return nil
}
