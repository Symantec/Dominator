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

type addressPoolType struct {
	Free       []proto.Address
	Registered []proto.Address
}

type DhcpServer interface {
	AddLease(address proto.Address, hostname string)
	AddSubnet(subnet proto.Subnet)
	MakeAcknowledgmentChannel(ipAddr net.IP) <-chan struct{}
	MakeRequestChannel(macAddr string) <-chan net.IP
	RemoveLease(ipAddr net.IP)
	RemoveSubnet(subnetId string)
}

type Manager struct {
	StartOptions
	importCookie      []byte
	memTotalInMiB     uint64
	numCPU            int
	serialNumber      string
	volumeDirectories []string
	mutex             sync.RWMutex // Lock everthing below (those can change).
	addressPool       addressPoolType
	healthStatus      string
	notifiers         map[<-chan proto.Update]chan<- proto.Update
	ownerGroups       map[string]struct{}
	ownerUsers        map[string]struct{}
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
	VlanIdToBridge     map[uint]string // Key: VLAN ID, value: bridge interface.
	VolumeDirectories  []string
}

type vmInfoType struct {
	mutex                      sync.RWMutex
	accessToken                []byte
	accessTokenCleanupNotifier chan<- struct{}
	proto.VmInfo
	VolumeLocations  []volumeType
	manager          *Manager
	dirname          string
	doNotWriteOrSend bool
	hasHealthAgent   bool
	ipAddress        string
	monitorSockname  string
	ownerUsers       map[string]struct{}
	commandChannel   chan<- string
	logger           log.DebugLogger
	destroyTimer     *time.Timer
	metadataChannels map[chan<- string]struct{}
	stoppedNotifier  chan<- struct{}
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
	return m.addAddressesToPool(addresses)
}

func (m *Manager) BecomePrimaryVmOwner(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.becomePrimaryVmOwner(ipAddr, authInfo)
}

func (m *Manager) ChangeOwners(ownerGroups, ownerUsers []string) error {
	return m.changeOwners(ownerGroups, ownerUsers)
}

func (m *Manager) ChangeVmOwnerUsers(ipAddr net.IP,
	authInfo *srpc.AuthInformation, extraUsers []string) error {
	return m.changeVmOwnerUsers(ipAddr, authInfo, extraUsers)
}

func (m *Manager) ChangeVmTags(ipAddr net.IP, authInfo *srpc.AuthInformation,
	tgs tags.Tags) error {
	return m.changeVmTags(ipAddr, authInfo, tgs)
}

func (m *Manager) CheckOwnership(authInfo *srpc.AuthInformation) bool {
	return m.checkOwnership(authInfo)
}

func (m *Manager) CheckVmHasHealthAgent(ipAddr net.IP) (bool, error) {
	return m.checkVmHasHealthAgent(ipAddr)
}

func (m *Manager) CloseUpdateChannel(channel <-chan proto.Update) {
	m.closeUpdateChannel(channel)
}

func (m *Manager) CommitImportedVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.commitImportedVm(ipAddr, authInfo)
}

func (m *Manager) CreateVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	return m.createVm(conn, decoder, encoder)
}

func (m *Manager) DeleteVmVolume(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte, volumeIndex uint) error {
	return m.deleteVmVolume(ipAddr, authInfo, accessToken, volumeIndex)
}

func (m *Manager) DestroyVm(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte) error {
	return m.destroyVm(ipAddr, authInfo, accessToken)
}

func (m *Manager) DiscardVmAccessToken(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte) error {
	return m.discardVmAccessToken(ipAddr, authInfo, accessToken)
}

func (m *Manager) DiscardVmOldImage(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.discardVmOldImage(ipAddr, authInfo)
}

func (m *Manager) DiscardVmOldUserData(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.discardVmOldUserData(ipAddr, authInfo)
}

func (m *Manager) DiscardVmSnapshot(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.discardVmSnapshot(ipAddr, authInfo)
}

func (m *Manager) GetHealthStatus() string {
	return m.getHealthStatus()
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

func (m *Manager) GetVmAccessToken(ipAddr net.IP,
	authInfo *srpc.AuthInformation, lifetime time.Duration) ([]byte, error) {
	return m.getVmAccessToken(ipAddr, authInfo, lifetime)
}

func (m *Manager) GetVmInfo(ipAddr net.IP) (proto.VmInfo, error) {
	return m.getVmInfo(ipAddr)
}

func (m *Manager) GetVmUserData(ipAddr net.IP) (io.ReadCloser, error) {
	rc, _, err := m.getVmUserData(ipAddr,
		&srpc.AuthInformation{HaveMethodAccess: true},
		nil)
	return rc, err
}

func (m *Manager) GetVmUserDataRPC(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte) (
	io.ReadCloser, uint64, error) {
	return m.getVmUserData(ipAddr, authInfo, accessToken)
}

func (m *Manager) GetVmVolume(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	return m.getVmVolume(conn, decoder, encoder)
}

func (m *Manager) ImportLocalVm(authInfo *srpc.AuthInformation,
	request proto.ImportLocalVmRequest) error {
	return m.importLocalVm(authInfo, request)
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

func (m *Manager) ListVolumeDirectories() []string {
	return m.volumeDirectories
}

func (m *Manager) MakeSubnetChannel() <-chan proto.Subnet {
	return m.makeSubnetChannel()
}

func (m *Manager) MakeUpdateChannel() <-chan proto.Update {
	return m.makeUpdateChannel()
}

func (m *Manager) MigrateVm(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder) error {
	return m.migrateVm(conn, decoder, encoder)
}

func (m *Manager) NotifyVmMetadataRequest(ipAddr net.IP, path string) {
	m.notifyVmMetadataRequest(ipAddr, path)
}

func (m *Manager) PrepareVmForMigration(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte, enable bool) error {
	return m.prepareVmForMigration(ipAddr, authInfo, accessToken, enable)
}

func (m *Manager) RemoveExcessAddressesFromPool(maxFree map[string]uint) error {
	return m.removeExcessAddressesFromPool(maxFree)
}

func (m *Manager) RegisterVmMetadataNotifier(ipAddr net.IP,
	authInfo *srpc.AuthInformation, pathChannel chan<- string) error {
	return m.registerVmMetadataNotifier(ipAddr, authInfo, pathChannel)
}

func (m *Manager) ReplaceVmImage(conn *srpc.Conn, decoder srpc.Decoder,
	encoder srpc.Encoder, authInfo *srpc.AuthInformation) error {
	return m.replaceVmImage(conn, decoder, encoder, authInfo)
}

func (m *Manager) ReplaceVmUserData(ipAddr net.IP, reader io.Reader,
	size uint64, authInfo *srpc.AuthInformation) error {
	return m.replaceVmUserData(ipAddr, reader, size, authInfo)
}

func (m *Manager) RestoreVmFromSnapshot(ipAddr net.IP,
	authInfo *srpc.AuthInformation, forceIfNotStopped bool) error {
	return m.restoreVmFromSnapshot(ipAddr, authInfo, forceIfNotStopped)
}

func (m *Manager) RestoreVmImage(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.restoreVmImage(ipAddr, authInfo)
}

func (m *Manager) RestoreVmUserData(ipAddr net.IP,
	authInfo *srpc.AuthInformation) error {
	return m.restoreVmUserData(ipAddr, authInfo)
}

func (m *Manager) SnapshotVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	forceIfNotStopped, snapshotRootOnly bool) error {
	return m.snapshotVm(ipAddr, authInfo, forceIfNotStopped, snapshotRootOnly)
}

func (m *Manager) StartVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte, dhcpTimeout time.Duration) (
	bool, error) {
	return m.startVm(ipAddr, authInfo, accessToken, dhcpTimeout)
}

func (m *Manager) StopVm(ipAddr net.IP, authInfo *srpc.AuthInformation,
	accessToken []byte) error {
	return m.stopVm(ipAddr, authInfo, accessToken)
}

func (m *Manager) UpdateSubnets(request proto.UpdateSubnetsRequest) error {
	return m.updateSubnets(request)
}

func (m *Manager) UnregisterVmMetadataNotifier(ipAddr net.IP,
	pathChannel chan<- string) error {
	return m.unregisterVmMetadataNotifier(ipAddr, pathChannel)
}

func (m *Manager) WriteHtml(writer io.Writer) {
	m.writeHtml(writer)
}
