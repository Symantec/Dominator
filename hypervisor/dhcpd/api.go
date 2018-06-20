package dhcpd

import (
	"net"
	"sync"

	"github.com/Symantec/Dominator/lib/log"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type DhcpServer struct {
	logger          log.DebugLogger
	myIP            net.IP
	mutex           sync.RWMutex
	ackChannels     map[string]chan struct{} // Key: IPaddr.
	ipAddrToMacAddr map[string]string        // Key: IPaddr, V: MACaddr.
	leases          map[string]leaseType     // Key: MACaddr.
	requestChannels map[string]chan net.IP   // Key: MACaddr.
	subnets         []proto.Subnet
}

type leaseType struct {
	proto.Address
	Hostname string
}

func New(bridges []string, logger log.DebugLogger) (*DhcpServer, error) {
	return newServer(bridges, logger)
}

func (s *DhcpServer) AddLease(address proto.Address, hostname string) {
	s.addLease(address, hostname)
}

func (s *DhcpServer) AddSubnet(subnet proto.Subnet) {
	s.addSubnet(subnet)
}

func (s *DhcpServer) MakeAcknowledgmentChannel(ipAddr net.IP) <-chan struct{} {
	return s.makeAcknowledgmentChannel(ipAddr)
}

func (s *DhcpServer) MakeRequestChannel(macAddr string) <-chan net.IP {
	return s.makeRequestChannel(macAddr)
}

func (s *DhcpServer) RemoveLease(ipAddr net.IP) {
	s.removeLease(ipAddr)
}

func (s *DhcpServer) RemoveSubnet(subnetId string) {
	s.removeSubnet(subnetId)
}
