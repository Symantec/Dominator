package client

import (
	"github.com/Symantec/Dominator/lib/hash"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"reflect"
)

func buildMachine(machine Machine) *machineType {
	computedFiles := make(map[string]string, len(machine.ComputedFiles))
	for _, computedFile := range machine.ComputedFiles {
		computedFiles[computedFile.Pathname] = computedFile.Source
	}
	return &machineType{
		machine:       machine.Machine,
		computedFiles: computedFiles}
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
		if sendRequests {
			m.sendYieldRequests(mapMachine)
		}
	}
}

func (m *Manager) sendYieldRequests(machine *machineType) {
	connectionMap := make(map[string][]string)
	for pathname, sourceName := range machine.computedFiles {
		connectionMap[sourceName] = append(connectionMap[sourceName], pathname)
	}
	for sourceName, pathnames := range connectionMap {
		source, ok := m.sourceMap[sourceName]
		if !ok {
			source = new(sourceType)
			sendChannel := make(chan proto.ClientRequest, 4096)
			source.sendChannel = sendChannel
			m.sourceMap[sourceName] = source
			go sendClientRequests(sourceName, sendChannel,
				m.serverMessageChannel, m.logger)
		}
		var request proto.ClientRequest
		request.YieldRequest = &proto.YieldRequest{machine.machine, pathnames}
		source.sendChannel <- request
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
			var request proto.ClientRequest
			request.GetObjectRequest = &proto.GetObjectRequest{file.Hash}
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
