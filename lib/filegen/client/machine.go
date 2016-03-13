package client

import (
	"github.com/Symantec/Dominator/lib/hash"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"reflect"
)

func buildMachine(machine Machine) *machineType {
	computedFiles := make(map[string]string, len(machine.ComputedFiles))
	sourceToPaths := make(map[string][]string)
	for _, computedFile := range machine.ComputedFiles {
		computedFiles[computedFile.Pathname] = computedFile.Source
		pathnames := sourceToPaths[computedFile.Source]
		pathnames = append(pathnames, computedFile.Pathname)
		sourceToPaths[computedFile.Source] = pathnames
	}
	return &machineType{
		machine:       machine.Machine,
		computedFiles: computedFiles,
		sourceToPaths: sourceToPaths}
}

func (m *Manager) addMachine(machine *machineType) {
	hostname := machine.machine.Hostname
	_, ok := m.machineMap[hostname]
	if ok {
		panic(hostname + ": already added")
	}
	m.machineMap[hostname] = machine
	m.sendYieldRequests(machine)
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
	if mapMachine, ok := m.machineMap[hostname]; !ok {
		panic(hostname + ": not present")
	} else {
		sendRequests := false
		if machine.machine != mapMachine.machine {
			mapMachine.machine = machine.machine
			sendRequests = true
		}
		if !reflect.DeepEqual(machine.computedFiles, mapMachine.computedFiles) {
			sendRequests = true
			mapMachine.computedFiles = machine.computedFiles
		}
		if !reflect.DeepEqual(machine.sourceToPaths, mapMachine.sourceToPaths) {
			sendRequests = true
			mapMachine.sourceToPaths = machine.sourceToPaths
		}
		if sendRequests {
			m.sendYieldRequests(mapMachine)
		}
	}
}

func (m *Manager) sendYieldRequests(machine *machineType) {
	for sourceName, pathnames := range machine.sourceToPaths {
		if source, existing := m.getSource(sourceName); existing {
			request := &proto.ClientRequest{
				YieldRequest: &proto.YieldRequest{machine.machine, pathnames}}
			source.sendChannel <- request
		}
	}
}

func (m *Manager) handleYieldResponse(machine *machineType,
	files []proto.FileInfo) {
	objectsToWaitFor := make(map[hash.Hash]struct{})
	waiterChannel := make(chan hash.Hash)
	for _, file := range files {
		sourceName, ok := machine.computedFiles[file.Pathname]
		if !ok {
			m.logger.Printf("no source name for: %s on: %s\n",
				file.Pathname, machine.machine.Hostname)
			continue
		}
		source, ok := m.sourceMap[sourceName]
		if !ok {
			panic("no source for: " + sourceName)
		}
		hashes := make([]hash.Hash, 1)
		hashes[0] = file.Hash
		if lengths, err := m.objectServer.CheckObjects(hashes); err != nil {
			panic(err)
		} else if lengths[0] < 1 {
			request := &proto.ClientRequest{
				GetObjectRequest: &proto.GetObjectRequest{file.Hash}}
			source.sendChannel <- request
			objectsToWaitFor[file.Hash] = struct{}{}
			if _, ok := m.objectWaiters[file.Hash]; !ok {
				m.objectWaiters[file.Hash] = make([]chan<- hash.Hash, 0, 1)
			}
			m.objectWaiters[file.Hash] = append(m.objectWaiters[file.Hash],
				waiterChannel)
		}
	}
	if len(objectsToWaitFor) > 0 {
		go waitForObjectsAndSendUpdate(waiterChannel, objectsToWaitFor,
			machine.updateChannel, files)
	} else {
		machine.updateChannel <- files
	}
}

func waitForObjectsAndSendUpdate(objectChannel <-chan hash.Hash,
	objectsToWaitFor map[hash.Hash]struct{},
	updateChannel chan<- []proto.FileInfo, files []proto.FileInfo) {
	defer func() {
		recover() // If updateChannel is closed, it means the machine went away.
	}()
	for hashVal := range objectChannel {
		delete(objectsToWaitFor, hashVal)
		if len(objectsToWaitFor) < 1 {
			updateChannel <- files // This will panic if the machine went away.
		}
	}
}
