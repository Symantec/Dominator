package client

import (
	"reflect"
)

func buildMachine(machine Machine) *machineType {
	computedFiles := make(map[string]string, len(machine.ComputedFiles))
	for _, computedFile := range machine.ComputedFiles {
		computedFiles[computedFile.Pathname] = computedFile.Source
	}
	return &machineType{
		machine:       machine.machine,
		computedFiles: computedFiles}
}

func (m *Manager) addMachine(machine *machineType) {
	hostname := machine.machine.Hostname
	_, ok := m.machineMap[hostname]
	if ok {
		panic(hostname + ": already added")
	}
	m.machineMap[hostname] = machine
}

func (m *Manager) removeMachine(hostname string) {
	if machine, ok := m.machineMap[hostname]; !ok {
		panic(hostname + ": not present")
	} else {
		delete(m.machineMap, hostname)
		close(machine.updateChannel)
	}
}

func (m *Manager) updateMachine(machine *machineType) {
	hostname := machine.machine.Hostname
	if oldMachine, ok := m.machineMap[hostname]; !ok {
		panic(hostname + ": not present")
	} else {
		sendRequest := false
		if machine.machine != oldMachine.machine {
			sendRequest = true
		}
		if !reflect.DeepEqual(machine.computedFiles, oldMachine.computedFiles) {
			sendRequest = true
			machine.computedFiles = oldMachine.computedFiles
		}
		_ = sendRequest // HACK
	}
}
