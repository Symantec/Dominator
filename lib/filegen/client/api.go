package client

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"log"
)

type ComputedFile struct {
	Pathname string
	Source   string
}

type Machine struct {
	machine       mdb.Machine
	ComputedFiles []ComputedFile
}

type machineType struct {
	machine       mdb.Machine
	updateChannel chan<- []proto.FileInfo
	computedFiles map[string]string // source = map[pathname]
}

type connection struct {
	client *srpc.Client
	conn   *srpc.Conn
}

type Manager struct {
	connMap              map[string]*connection
	objectServer         objectserver.ObjectServer
	machineMap           map[string]*machineType
	addMachineChannel    chan *machineType
	removeMachineChannel chan string
	updateMachineChannel chan *machineType
	logger               *log.Logger
}

// New creates a new *Manager. Only one should be created per application.
// The logger will be used to log problems.
func New(objSrv objectserver.ObjectServer, logger *log.Logger) *Manager {
	return newManager(objSrv, logger)
}

// Add will add a machine to the Manager. Re-adding a machine will result in a
// panic.
func (m *Manager) Add(machine Machine) <-chan []proto.FileInfo {
	updateChannel := make(chan []proto.FileInfo)
	mach := buildMachine(machine)
	mach.updateChannel = updateChannel
	m.addMachineChannel <- mach
	return updateChannel
}

// Remove will remove a machine from the Manager.
func (m *Manager) Remove(hostname string) {
	m.removeMachineChannel <- hostname
}

// Update will update the machine data for a machine.
func (m *Manager) Update(machine Machine) {
	m.updateMachineChannel <- buildMachine(machine)
}
