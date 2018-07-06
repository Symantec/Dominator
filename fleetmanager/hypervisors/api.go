package hypervisors

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/Symantec/Dominator/fleetmanager/topology"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	fm_proto "github.com/Symantec/Dominator/proto/fleetmanager"
	hyper_proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const (
	probeStatusNotYetProbed probeStatus = iota
	probeStatusGood
	probeStatusNoSrpc
	probeStatusNoService
	probeStatusBad
)

type hypervisorType struct {
	logger          log.DebugLogger
	mutex           sync.RWMutex
	conn            *srpc.Conn
	deleteScheduled bool
	machine         *fm_proto.Machine
	probeStatus     probeStatus
	subnets         []hyper_proto.Subnet
	vms             map[string]*vmInfoType // Key: VM IP address.
}

type IpStorer interface {
	AddIPsForHypervisor(hypervisor net.IP, addrs []net.IP) error
	CheckIpIsRegistered(addr net.IP) (bool, error)
	SetIPsForHypervisor(hypervisor net.IP, addrs []net.IP) error
	UnregisterHypervisor(hypervisor net.IP) error
}

type Manager struct {
	storer      Storer
	logger      log.DebugLogger
	invertTable [256]byte
	mutex       sync.RWMutex
	topology    *topology.Topology
	hypervisors map[string]*hypervisorType // Key: hypervisor machine name.
	subnets     map[string]*subnetType     // Key: Gateway IP.
	vms         map[string]*vmInfoType     // Key: VM IP address.
}

type probeStatus uint

type Storer interface {
	IpStorer
	VmStorer
}

type subnetType struct {
	subnet  *topology.Subnet
	startIp net.IP
	stopIp  net.IP
	nextIp  net.IP
}

type vmInfoType struct {
	ipAddr string
	hyper_proto.VmInfo
	hypervisor *hypervisorType
}

type VmStorer interface {
	DeleteVm(hypervisor net.IP, ipAddr string) error
	ListVMs(hypervisor net.IP) ([]string, error)
	ReadVm(hypervisor net.IP, ipAddr string) (*hyper_proto.VmInfo, error)
	WriteVm(hypervisor net.IP, ipAddr string, vmInfo hyper_proto.VmInfo) error
}

func New(storer Storer, logger log.DebugLogger) (*Manager, error) {
	if err := checkPoolLimits(); err != nil {
		return nil, err
	}
	manager := &Manager{
		storer:      storer,
		logger:      logger,
		hypervisors: make(map[string]*hypervisorType),
		subnets:     make(map[string]*subnetType),
		vms:         make(map[string]*vmInfoType),
	}
	manager.initInvertTable()
	http.HandleFunc("/listHypervisors", manager.listHypervisorsHandler)
	http.HandleFunc("/listLocations", manager.listLocationsHandler)
	http.HandleFunc("/listVMs", manager.listVMsHandler)
	return manager, nil
}

func (m *Manager) GetHypervisorForVm(ipAddr net.IP) (string, error) {
	return m.getHypervisorForVm(ipAddr)
}

func (m *Manager) ListHypervisorsInLocation(
	request fm_proto.ListHypervisorsInLocationRequest) ([]string, error) {
	return m.listHypervisorsInLocation(request)
}

func (m *Manager) ListLocations(dirname string) ([]string, error) {
	return m.listLocations(dirname)
}

func (m *Manager) ListVMsInLocation(dirname string) ([]net.IP, error) {
	return m.listVMsInLocation(dirname)
}

func (m *Manager) WriteHtml(writer io.Writer) {
	m.writeHtml(writer)
}

func (m *Manager) UpdateTopology(t *topology.Topology) {
	m.updateTopology(t)
}
