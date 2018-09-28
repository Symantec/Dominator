package fsstorer

import (
	"net"
	"sync"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/tags"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type IP [4]byte

type Storer struct {
	topDir          string
	logger          log.DebugLogger
	mutex           sync.RWMutex
	hypervisorToIPs map[IP][]IP // Key: hypervisor IP address.
	ipToHypervisor  map[IP]IP   // Key: IP address, value: hypervisor.
}

func New(topDir string, logger log.DebugLogger) (*Storer, error) {
	storer := &Storer{
		topDir:          topDir,
		logger:          logger,
		hypervisorToIPs: make(map[IP][]IP),
		ipToHypervisor:  make(map[IP]IP),
	}
	if err := storer.load(); err != nil {
		return nil, err
	}
	return storer, nil
}

func (ip IP) String() string {
	return ip.string()
}

func (s *Storer) AddIPsForHypervisor(hypervisor net.IP,
	addrs []net.IP) error {
	return s.addIPsForHypervisor(hypervisor, addrs)
}

func (s *Storer) CheckIpIsRegistered(addr net.IP) (bool, error) {
	return s.checkIpIsRegistered(addr)
}

func (s *Storer) DeleteVm(hypervisor net.IP, ipAddr string) error {
	return s.deleteVm(hypervisor, ipAddr)
}

func (s *Storer) ListVMs(hypervisor net.IP) ([]string, error) {
	return s.listVMs(hypervisor)
}

func (s *Storer) ReadMachineTags(hypervisor net.IP) (tags.Tags, error) {
	return s.readMachineTags(hypervisor)
}

func (s *Storer) ReadVm(hypervisor net.IP,
	ipAddr string) (*proto.VmInfo, error) {
	return s.readVm(hypervisor, ipAddr)
}

func (s *Storer) SetIPsForHypervisor(hypervisor net.IP,
	addrs []net.IP) error {
	return s.setIPsForHypervisor(hypervisor, addrs)
}

func (s *Storer) UnregisterHypervisor(hypervisor net.IP) error {
	return s.unregisterHypervisor(hypervisor)
}

func (s *Storer) WriteMachineTags(hypervisor net.IP, tgs tags.Tags) error {
	return s.writeMachineTags(hypervisor, tgs)
}

func (s *Storer) WriteVm(hypervisor net.IP, ipAddr string,
	vmInfo proto.VmInfo) error {
	return s.writeVm(hypervisor, ipAddr, vmInfo)
}
