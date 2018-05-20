package hypervisors

import (
	"io"
	"net"
	"net/http"
	"sync"

	"github.com/Symantec/Dominator/fleetmanager/topology"
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

const (
	probeStatusNotYetProbed = iota
	probeStatusGood
	probeStatusBad
)

type hypervisorType struct {
	logger          log.DebugLogger
	mutex           sync.RWMutex
	conn            *srpc.Conn
	deleteScheduled bool
	machine         *topology.Machine
	probeStatus     probeStatus
}

type IpStorer interface {
	AddIPsForHypervisor(hypervisor net.IP, addrs []net.IP) error
	CheckIpIsRegistered(addr net.IP) (bool, error)
	SetIPsForHypervisor(hypervisor net.IP, addrs []net.IP) error
	UnregisterHypervisor(hypervisor net.IP) error
}

type Manager struct {
	ipStorer    IpStorer
	logger      log.DebugLogger
	invertTable [256]byte
	mutex       sync.RWMutex
	topology    *topology.Topology
	hypervisors map[string]*hypervisorType // Key: hypervisor machine name.
	subnets     map[string]*subnetType     // Key: Gateway IP.
	vms         map[string]*vmInfoType     // Key: VM IP address.
}

type probeStatus uint

type subnetType struct {
	subnet  *topology.Subnet
	startIp net.IP
	stopIp  net.IP
	nextIp  net.IP
}

type vmInfoType struct {
	ipAddr string
	proto.VmInfo
	hypervisor *hypervisorType
}

func New(ipStorer IpStorer, logger log.DebugLogger) (*Manager, error) {
	if err := checkPoolLimits(); err != nil {
		return nil, err
	}
	manager := &Manager{
		ipStorer:    ipStorer,
		logger:      logger,
		hypervisors: make(map[string]*hypervisorType),
		subnets:     make(map[string]*subnetType),
		vms:         make(map[string]*vmInfoType),
	}
	manager.initInvertTable()
	http.HandleFunc("/listHypervisors", manager.listHypervisorsHandler)
	http.HandleFunc("/listVMs", manager.listVMsHandler)
	return manager, nil
}

func (m *Manager) GetHypervisorForVm(ipAddr net.IP) (string, error) {
	return m.getHypervisorForVm(ipAddr)
}

func (m *Manager) WriteHtml(writer io.Writer) {
	m.writeHtml(writer)
}

func (m *Manager) UpdateTopology(t *topology.Topology) {
	m.updateTopology(t)
}
