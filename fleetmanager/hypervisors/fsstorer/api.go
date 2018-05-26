package fsstorer

import (
	"net"
	"sync"

	"github.com/Symantec/Dominator/lib/log"
)

type IP [4]byte

type IpStorer struct {
	topDir          string
	logger          log.DebugLogger
	mutex           sync.RWMutex
	hypervisorToIPs map[IP][]IP // Key: hypervisor IP address.
	ipToHypervisor  map[IP]IP   // Key: IP address, value: hypervisor.
}

func New(topDir string, logger log.DebugLogger) (*IpStorer, error) {
	ipStorer := &IpStorer{
		topDir:          topDir,
		logger:          logger,
		hypervisorToIPs: make(map[IP][]IP),
		ipToHypervisor:  make(map[IP]IP),
	}
	if err := ipStorer.load(); err != nil {
		return nil, err
	}
	return ipStorer, nil
}

func (s *IpStorer) AddIPsForHypervisor(hypervisor net.IP,
	addrs []net.IP) error {
	return s.addIPsForHypervisor(hypervisor, addrs)
}

func (s *IpStorer) CheckIpIsRegistered(addr net.IP) (bool, error) {
	return s.checkIpIsRegistered(addr)
}

func (s *IpStorer) SetIPsForHypervisor(hypervisor net.IP,
	addrs []net.IP) error {
	return s.setIPsForHypervisor(hypervisor, addrs)
}

func (s *IpStorer) UnregisterHypervisor(hypervisor net.IP) error {
	return s.unregisterHypervisor(hypervisor)
}
