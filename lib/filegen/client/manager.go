package client

import (
	"github.com/Symantec/Dominator/lib/objectserver"
	"log"
)

func newManager(objSrv objectserver.ObjectServer, logger *log.Logger) *Manager {
	m := &Manager{
		machineMap:           make(map[string]*machineType),
		addMachineChannel:    make(chan *machineType),
		removeMachineChannel: make(chan string),
		updateMachineChannel: make(chan *machineType),
		objectServer:         objSrv,
		logger:               logger}
	go m.manage()
	return m
}

func (m *Manager) manage() {
	for {
		select {
		case machine := <-m.addMachineChannel:
			m.addMachine(machine)
		case hostname := <-m.removeMachineChannel:
			m.removeMachine(hostname)
		case machine := <-m.updateMachineChannel:
			m.updateMachine(machine)
		}
	}
}
