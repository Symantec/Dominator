package manager

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/lib/tags"
	proto "github.com/Symantec/Dominator/proto/hypervisor"
)

type DhcpServer interface {
	AddLease(address proto.Address)
	AddSubnet(subnet proto.Subnet)
	MakeAcknowledgmentChannel(ipAddr net.IP) <-chan struct{}
	RemoveLease(ipAddr net.IP)
}

type Manager struct {
	StartOptions
	numCPU            int
	memTotalInMiB     uint64
	volumeDirectories []string
	mutex             sync.RWMutex // Lock everthing below (those can change).
	addressPool       []proto.Address
	subnets           map[string]proto.Subnet // Key: Subnet ID.
	subnetChannels    []chan<- proto.Subnet
	vms               map[string]*vmInfoType // Key: IP address.
}

type StartOptions struct {
	DhcpServer         DhcpServer
	ImageServerAddress string
	Logger             log.DebugLogger
	ShowVgaConsole     bool
	StateDir           string
	Username           string
	VolumeDirectories  []string
}

type vmInfoType struct {
	mutex sync.Mutex
	proto.VmInfo
	VolumeLocations []volumeType
	manager         *Manager
	dirname         string
	hasHealthAgent  bool
	monitorSockname string
	ownerUsers      map[string]struct{}
	commandChannel  chan<- string
	logger          log.DebugLogger
	destroyTimer    *time.Timer
}

type volumeType struct {
	DirectoryToCleanup string
	Filename           string
}

func New(startOptions StartOptions) (*Manager, error) {
	return newManager(startOptions)
}

func (m *Manager) AcknowledgeVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.acknowledgeVm(ipAddr, authInfo)
}

func (m *Manager) AddAddressesToPool(addresses []proto.Address) error {
	return m.addAddressesToPool(addresses, true)
}

func (m *Manager) AddSubnets(subnets []proto.Subnet) error {
	return m.addSubnets(subnets)
}

func (m *Manager) ChangeVmTags(ipAddr net.IP, authInfo *srpc.AuthInformation,
	tgs tags.Tags) error {
	return m.changeVmTags(ipAddr, authInfo, tgs)
}

func (m *Manager) CheckVmHasHealthAgent(ipAddr net.IP) (bool, error) {
	return m.checkVmHasHealthAgent(ipAddr)
}

func (m *Manager) CreateVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	return m.createVm(conn, decoder, encoder)
}

func (m *Manager) DestroyVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.destroyVm(ipAddr, authInfo)
}

func (m *Manager) GetImageServerAddress() string {
	return m.ImageServerAddress
}

func (m *Manager) GetNumVMs() (uint, uint) {
	return m.getNumVMs()
}

func (m *Manager) GetVmBootLog(ipAddr net.IP) (io.ReadCloser, error) {
	return m.getVmBootLog(ipAddr)
}

func (m *Manager) GetVmInfo(ipAddr net.IP) (proto.VmInfo, error) {
	return m.getVmInfo(ipAddr)
}

func (m *Manager) GetVmUserData(ipAddr net.IP) (io.ReadCloser, error) {
	return m.getVmUserData(ipAddr)
}

func (m *Manager) ListAvailableAddresses() []proto.Address {
	return m.listAvailableAddresses()
}

func (m *Manager) ListSubnets(doSort bool) []proto.Subnet {
	return m.listSubnets(doSort)
}

func (m *Manager) ListVMs(doSort bool) []string {
	return m.listVMs(doSort)
}

func (m *Manager) MakeSubnetChannel() <-chan proto.Subnet {
	return m.makeSubnetChannel()
}

func (m *Manager) ReplaceVmImage(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder, authInfo *srpc.AuthInformation) error {
	return m.replaceVmImage(conn, decoder, encoder, authInfo)
}

func (m *Manager) RestoreVmImage(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.restoreVmImage(ipAddr, authInfo)
}

func (m *Manager) StartVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	dhcpTimeout time.Duration) (
	bool, error) {
	return m.startVm(ipAddr, authInfo, dhcpTimeout)
}

func (m *Manager) StopVm(ipAddr net.IP, authInfo *srpc.AuthInformation) error {
	return m.stopVm(ipAddr, authInfo)
}

func (m *Manager) WriteHtml(writer io.Writer) {
	m.writeHtml(writer)
}
