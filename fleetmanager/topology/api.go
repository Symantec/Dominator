package topology

import (
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/proto/hypervisor"
)

type Directory struct {
	Name             string
	Directories      []*Directory          `json:",omitempty"`
	Machines         []*Machine            `json:",omitempty"`
	Subnets          []*Subnet             `json:",omitempty"`
	nameToDirectory  map[string]*Directory // Key: directory name.
	parent           *Directory
	path             string
	subnetIdToSubnet map[string]*Subnet // Key: subnet ID.
}

func (directory *Directory) GetPath() string {
	return directory.path
}

func (directory *Directory) Walk(fn func(*Directory) error) error {
	return directory.walk(fn)
}

type HardwareAddr net.HardwareAddr

type Machine struct {
	Hostname       string       `json:",omitempty"`
	HostIpAddress  net.IP       `json:",omitempty"`
	HostMacAddress HardwareAddr `json:",omitempty"`
}

type Subnet struct {
	hypervisor.Subnet
	ReservedIPs     []net.IP            `json:",omitempty"`
	reservedIpAddrs map[string]struct{} // Key: IP address.
}

func (s *Subnet) GetReservedIpSet() map[string]struct{} {
	return s.reservedIpAddrs
}

func (subnet *Subnet) Shrink() {
	subnet.shrink()
}

type Topology struct {
	Root           *Directory
	machineParents map[string]*Directory // Key: machine name.
}

func Load(topologyDir string) (*Topology, error) {
	return load(topologyDir)
}

func Watch(topologyDir string, checkInterval time.Duration,
	logger log.DebugLogger) (<-chan *Topology, error) {
	return watch(topologyDir, checkInterval, logger)
}

func (t *Topology) FindDirectory(dirname string) (*Directory, error) {
	return t.findDirectory(dirname)
}

func (t *Topology) GetLocationOfMachine(name string) (string, error) {
	return t.getLocationOfMachine(name)
}

func (t *Topology) GetNumMachines() uint {
	return uint(len(t.machineParents))
}

func (t *Topology) GetSubnetsForMachine(name string) ([]*Subnet, error) {
	return t.getSubnetsForMachine(name)
}

func (t *Topology) ListMachines(dirname string) ([]*Machine, error) {
	return t.listMachines(dirname)
}

func (t *Topology) Walk(fn func(*Directory) error) error {
	return t.Root.Walk(fn)
}
