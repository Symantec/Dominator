package filegen

import (
	"bytes"
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io/ioutil"
	"time"
)

func (t *rpcType) Connect(conn *srpc.Conn) error {
	return t.manager.connect(conn) // Long-lived.
}

func (m *Manager) connect(conn *srpc.Conn) error {
	defer conn.Flush()
	notifierChannel := make(chan notificationData, 1)
	m.rwMutex.Lock()
	m.notifiers[notifierChannel] = notifierChannel
	m.rwMutex.Unlock()
	defer func() {
		m.rwMutex.Lock()
		delete(m.notifiers, notifierChannel)
		m.rwMutex.Unlock()
	}()
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	for ; ; conn.Flush() {
		var request proto.ClientRequest
		var serverMessage proto.ServerMessage
		if err := decoder.Decode(&request); err != nil {
			return err
		}
		if request := request.YieldRequest; request != nil {
			m.updateMachineData(request.Machine)
			fileInfos := make([]proto.FileInfo, len(request.Pathnames))
			serverMessage.YieldResponse = &proto.YieldResponse{
				Hostname: request.Machine.Hostname,
				Files:    fileInfos}
			for index, pathname := range request.Pathnames {
				fileInfos[index] = m.computeFile(request.Machine, pathname)
			}
		}
		if request := request.GetObjectRequest; request != nil {
			_, reader, err := m.objectServer.GetObject(request.Hash)
			if err != nil {
				m.logger.Println(err)
			} else {
				data, _ := ioutil.ReadAll(reader)
				serverMessage.GetObjectResponse = &proto.GetObjectResponse{
					Hash: request.Hash,
					Data: data}
				reader.Close()
			}
		}
		return encoder.Encode(serverMessage)
	}
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
	pathname string) proto.FileInfo {
	fileInfo := proto.FileInfo{Pathname: pathname}
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
			fileInfo.Hash = fi.hash
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
	fileInfo.Hash = hashVal
	fileInfo.ValidUntil = validUntil
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()
	pathMgr.machineHashes[machine.Hostname] = expiringHash{hashVal, validUntil}
	return fileInfo
}
