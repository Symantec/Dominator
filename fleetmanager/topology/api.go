package topology

import (
	"net"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/tags"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type Directory struct {
	Name             string
	Directories      []*Directory          `json:",omitempty"`
	Machines         []*fm_proto.Machine   `json:",omitempty"`
	Subnets          []*Subnet             `json:",omitempty"`
	Tags             tags.Tags             `json:",omitempty"`
	nameToDirectory  map[string]*Directory // Key: directory name.
	owners           *ownersType
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

type ownersType struct {
	OwnerGroups []string `json:",omitempty"`
	OwnerUsers  []string `json:",omitempty"`
}

type Subnet struct {
	hyper_proto.Subnet
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
	Root            *Directory
	machineParents  map[string]*Directory // Key: machine name.
	reservedIpAddrs map[string]struct{}   // Key: IP address.
}

func Load(topologyDir string) (*Topology, error) {
	return load(topologyDir)
}

func Watch(topologyRepository, localRepositoryDir, topologyDir string,
	checkInterval time.Duration,
	logger log.DebugLogger) (<-chan *Topology, error) {
	return watch(topologyRepository, localRepositoryDir, topologyDir,
		checkInterval, logger)
}

func (t *Topology) CheckIfIpIsReserved(ipAddr string) bool {
	_, ok := t.reservedIpAddrs[ipAddr]
	return ok
}

func (t *Topology) CheckIfMachineHasSubnet(name, subnetId string) (
	bool, error) {
	return t.checkIfMachineHasSubnet(name, subnetId)
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

func (t *Topology) ListMachines(dirname string) ([]*fm_proto.Machine, error) {
	return t.listMachines(dirname)
}

func (t *Topology) Walk(fn func(*Directory) error) error {
	return t.Root.Walk(fn)
}
