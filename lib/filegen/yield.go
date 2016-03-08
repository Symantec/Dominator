package filegen

import (
	"bytes"
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/filegenerator"
	"time"
)

func (t *rpcType) Yield(conn *srpc.Conn) error {
	return t.manager.yield(conn)
}

func (m *Manager) yield(conn *srpc.Conn) error {
	defer conn.Flush()
	var request filegenerator.YieldRequest
	decoder := gob.NewDecoder(conn)
	if err := decoder.Decode(&request); err != nil {
		_, err = conn.WriteString(err.Error() + "\n")
		return err
	}
	if _, err := conn.WriteString("\n"); err != nil {
		return err
	}
	m.updateMachineData(request.Machine)
	encoder := gob.NewEncoder(conn)
	var response filegenerator.YieldResponse
	response.Hostname = request.Machine.Hostname
	response.Files = make([]filegenerator.FileInfo, len(request.Pathnames))
	for index, pathname := range request.Pathnames {
		response.Files[index] = m.computeFile(request.Machine, pathname)
	}
	return encoder.Encode(response)
}

func (m *Manager) updateMachineData(machine mdb.Machine) {
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()
	if oldMachine, ok := m.machineData[machine.Hostname]; !ok {
		m.machineData[machine.Hostname] = machine
	} else if oldMachine != machine {
		m.machineData[machine.Hostname] = machine
		for _, pathMgr := range m.pathManagers {
			delete(pathMgr.machineHashes, machine.Hostname)
		}
	}
}

func (m *Manager) computeFile(machine mdb.Machine,
	pathname string) filegenerator.FileInfo {
	fileInfo := filegenerator.FileInfo{Pathname: pathname}
	m.rwMutex.RLock()
	pathMgr, ok := m.pathManagers[pathname]
	if !ok {
		m.rwMutex.RUnlock()
		m.logger.Println("no generator for: " + pathname)
		return fileInfo
	}
	if fi, ok := pathMgr.machineHashes[machine.Hostname]; ok {
		if !fi.validUntil.IsZero() && time.Now().Before(fi.validUntil) {
			m.rwMutex.RUnlock()
			fileInfo.Hash = &fi.hash
			fileInfo.ValidUntil = fi.validUntil
			return fileInfo
		}
	}
	m.rwMutex.RUnlock()
	data, validUntil, err := pathMgr.generator.Generate(machine, m.logger)
	if err != nil {
		return fileInfo
	}
	hashVal, _, err := m.objectServer.AddObject(bytes.NewReader(data),
		uint64(len(data)), nil)
	if err != nil {
		m.logger.Println(err)
		return fileInfo
	}
	fileInfo.Hash = &hashVal
	fileInfo.ValidUntil = validUntil
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()
	pathMgr.machineHashes[machine.Hostname] = expiringHash{hashVal, validUntil}
	return fileInfo
}
