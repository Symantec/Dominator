package rpcd

import (
	"fmt"
	"net"

	"github.com/Cloud-Foundations/Dominator/lib/errors"
	"github.com/Cloud-Foundations/Dominator/lib/srpc"
	"github.com/Cloud-Foundations/Dominator/proto/hypervisor"
)

func (t *srpcType) RegisterExternalLeases(conn *srpc.Conn,
	request hypervisor.RegisterExternalLeasesRequest,
	reply *hypervisor.RegisterExternalLeasesResponse) error {
	*reply = hypervisor.RegisterExternalLeasesResponse{
		errors.ErrorToString(t.registerExternalLeases(request))}
	return nil
}

func (t *srpcType) registerExternalLeases(
	request hypervisor.RegisterExternalLeasesRequest) error {
	hostnames := make(map[ipv4Address]string)
	leasesToRegister := make(map[ipv4Address]string)
	for index, address := range request.Addresses {
		if ip4 := address.IpAddress.To4(); ip4 == nil {
			return fmt.Errorf("%s is not an IPv4 address", address.IpAddress)
		} else {
			var addr [4]byte
			copy(addr[:], ip4)
			leasesToRegister[addr] = address.MacAddress
			if index < len(request.Hostnames) {
				hostnames[addr] = request.Hostnames[index]
			}
		}
	}
	leasesToDelete := make(map[ipv4Address]struct{})
	t.mutex.Lock()
	defer t.mutex.Unlock()
	for ipAddr := range t.externalLeases {
		ipAddr := ipAddr // Make a unique copy of the array.
		leasesToDelete[ipAddr] = struct{}{}
	}
	for ipAddr, macAddr := range leasesToRegister {
		if leasedMac, ok := t.externalLeases[ipAddr]; ok {
			if macAddr == leasedMac {
				delete(leasesToDelete, ipAddr)
				delete(leasesToRegister, ipAddr)
			}
		}
	}
	t.logger.Printf("RegisterExternalLeases: adding: %d, deleting: %d\n",
		len(leasesToRegister), len(leasesToDelete))
	for ipAddr := range leasesToDelete {
		ip := net.IP(ipAddr[:])
		t.logger.Debugf(1, "deleting: %s\n", ip)
		t.dhcpServer.RemoveLease(ip)
		delete(t.externalLeases, ipAddr)
	}
	for ipAddr, macAddr := range leasesToRegister {
		ipAddr := ipAddr // Make a unique copy of the array.
		addr := hypervisor.Address{IpAddress: ipAddr[:], MacAddress: macAddr}
		t.logger.Debugf(1, "adding: %s\n", addr)
		hostname := hostnames[ipAddr]
		if hostname == "" {
			hostname = addr.IpAddress.String()
		}
		if err := t.dhcpServer.AddLease(addr, hostname); err != nil {
			return err
		}
		t.externalLeases[ipAddr] = macAddr
	}
	return nil
}
