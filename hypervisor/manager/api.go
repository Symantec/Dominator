package manager

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/objectserver/cachingreader"
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
	rootCookie        []byte
	memTotalInMiB     uint64
	numCPU            int
	serialNumber      string
	volumeDirectories []string
	mutex             sync.RWMutex // Lock everything below (those can change).
	addressPool       addressPoolType
	healthStatus      string
	notifiers         map[<-chan proto.Update]chan<- proto.Update
	objectCache       *cachingreader.ObjectServer
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
	ObjectCacheBytes   uint64
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
	commandChannel             chan<- string
	destroyTimer               *time.Timer
	dirname                    string
	doNotWriteOrSend           bool
	hasHealthAgent             bool
	ipAddress                  string
	logger                     log.DebugLogger
	manager                    *Manager
	metadataChannels           map[chan<- string]struct{}
	monitorSockname            string
	ownerUsers                 map[string]struct{}
	serialInput                io.Writer
	serialOutput               chan<- byte
	stoppedNotifier            chan<- struct{}
	proto.LocalVmInfo
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

func (m *Manager) ChangeVmConsoleType(ipAddr net.IP,
	authInfo *srpc.AuthInformation, consoleType proto.ConsoleType) error {
	return m.changeVmConsoleType(ipAddr, authInfo, consoleType)
}

func (m *Manager) ChangeVmDestroyProtection(ipAddr net.IP,
	authInfo *srpc.AuthInformation, destroyProtection bool) error {
	return m.changeVmDestroyProtection(ipAddr, authInfo, destroyProtection)
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

func (m *Manager) ConnectToVmConsole(ipAddr net.IP,
	authInfo *srpc.AuthInformation) (net.Conn, error) {
	return m.connectToVmConsole(ipAddr, authInfo)
}

func (m *Manager) ConnectToVmSerialPort(ipAddr net.IP,
	authInfo *srpc.AuthInformation,
	portNumber uint) (chan<- byte, <-chan byte, error) {
	return m.connectToVmSerialPort(ipAddr, authInfo, portNumber)
}

func (m *Manager) CopyVm(conn *srpc.Conn, request proto.CopyVmRequest) error {
	return m.copyVm(conn, request)
}

func (m *Manager) CreateVm(conn *srpc.Conn) error {
	return m.createVm(conn)
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

func (m *Manager) ExportLocalVm(authInfo *srpc.AuthInformation,
	request proto.ExportLocalVmRequest) (*proto.ExportLocalVmInfo, error) {
	return m.exportLocalVm(authInfo, request)
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

func (m *Manager) GetVmVolume(conn *srpc.Conn) error {
	return m.getVmVolume(conn)
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

func (m *Manager) ListVMs(ownerUsers []string, doSort bool) []string {
	return m.listVMs(ownerUsers, doSort)
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

func (m *Manager) MigrateVm(conn *srpc.Conn) error {
	return m.migrateVm(conn)
}

func (m *Manager) NotifyVmMetadataRequest(ipAddr net.IP, path string) {
	m.notifyVmMetadataRequest(ipAddr, path)
}

func (m *Manager) PatchVmImage(conn *srpc.Conn,
	request proto.PatchVmImageRequest) error {
	return m.patchVmImage(conn, request)
}

func (m *Manager) PrepareVmForMigration(ipAddr net.IP,
	authInfo *srpc.AuthInformation, accessToken []byte, enable bool) error {
	return m.prepareVmForMigration(ipAddr, authInfo, accessToken, enable)
}

func (m *Manager) RemoveAddressesFromPool(addresses []proto.Address) error {
	return m.removeAddressesFromPool(addresses)
}

func (m *Manager) RemoveExcessAddressesFromPool(maxFree map[string]uint) error {
	return m.removeExcessAddressesFromPool(maxFree)
}

func (m *Manager) RegisterVmMetadataNotifier(ipAddr net.IP,
	authInfo *srpc.AuthInformation, pathChannel chan<- string) error {
	return m.registerVmMetadataNotifier(ipAddr, authInfo, pathChannel)
}

func (m *Manager) ReplaceVmImage(conn *srpc.Conn,
	authInfo *srpc.AuthInformation) error {
	return m.replaceVmImage(conn, authInfo)
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

func (m *Manager) ShutdownVMsAndExit() {
	m.shutdownVMsAndExit()
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
