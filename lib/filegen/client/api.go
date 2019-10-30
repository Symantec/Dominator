package client

import (
	"sync"

	"github.com/Cloud-Foundations/Dominator/lib/hash"
	"github.com/Cloud-Foundations/Dominator/lib/log"
	"github.com/Cloud-Foundations/Dominator/lib/mdb"
	"github.com/Cloud-Foundations/Dominator/lib/objectserver"
	proto "github.com/Cloud-Foundations/Dominator/proto/filegenerator"
)

type ComputedFile struct {
	Pathname string
	Source   string
}

type gauge struct {
	sync.Mutex
	value uint64
}

type Machine struct {
	Machine       mdb.Machine
	ComputedFiles []ComputedFile
}

type machineType struct {
	machine       mdb.Machine
	updateChannel chan<- []proto.FileInfo
	computedFiles map[string]string   // map[pathname] => source
	sourceToPaths map[string][]string // map[source] => []pathnames
}

type sourceType struct {
	sendChannel chan<- *proto.ClientRequest
}

type serverMessageType struct {
	source        string
	serverMessage proto.ServerMessage
}

type Manager struct {
	sourceMap              map[string]*sourceType
	objectServer           objectserver.ObjectServer
	machineMap             map[string]*machineType
	addMachineChannel      chan *machineType
	removeMachineChannel   chan string
	updateMachineChannel   chan *machineType
	serverMessageChannel   chan *serverMessageType
	sourceReconnectChannel chan<- string
	objectWaiters          map[hash.Hash][]chan<- hash.Hash
	numObjectWaiters       gauge
	logger                 log.Logger
}

// New creates a new *Manager. Object data will be added to the object server
// objSrv. Only one Manager should be created per application.
// The logger will be used to log problems.
func New(objSrv objectserver.ObjectServer, logger log.Logger) *Manager {
	return newManager(objSrv, logger)
}

// Add will add a machine to the Manager. Re-adding a machine will result in a
// panic. The length of the returned channel buffer is determined by size.
// A channel is returned from which file information may be read. It is
// guaranteed that corresponding object data are in the object server before
// file information is available.
func (m *Manager) Add(machine Machine, size uint) <-chan []proto.FileInfo {
	updateChannel := make(chan []proto.FileInfo, size)
	mach := buildMachine(machine)
	mach.updateChannel = updateChannel
	m.addMachineChannel <- mach
	return updateChannel
}

// Remove will remove a machine from the Manager. The corresponding file info
// channel will be closed.
func (m *Manager) Remove(hostname string) {
	m.removeMachineChannel <- hostname
}

// Update will update the machine data for a machine, which may result in file
// info data being sent to the corresponding channel.
func (m *Manager) Update(machine Machine) {
	m.updateMachineChannel <- buildMachine(machine)
}
