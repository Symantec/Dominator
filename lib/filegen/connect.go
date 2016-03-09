package filegen

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io/ioutil"
	"sync"
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
		close(notifierChannel)
	}()
	transmitLock := new(sync.Mutex)
	decoder := gob.NewDecoder(conn)
	encoder := gob.NewEncoder(conn)
	go m.handleNotifications(conn, decoder, encoder, transmitLock,
		notifierChannel)
	for ; ; conn.Flush() {
		if err := m.handleMessage(decoder, encoder, transmitLock); err != nil {
			return err
		}
	}
}

func (m *Manager) handleNotifications(conn *srpc.Conn, decoder *gob.Decoder,
	encoder *gob.Encoder, transmitLock *sync.Mutex,
	notificationChannel <-chan notificationData) {
	for notification := range notificationChannel {
		var serverMessage proto.ServerMessage
		serverMessage.InvalidateNotice = &proto.InvalidateNotice{
			Pathname: notification.pathname,
			Hostname: notification.hostname}
		transmitLock.Lock()
		encoder.Encode(serverMessage)
		transmitLock.Unlock()
		conn.Flush()
	}
}

func (m *Manager) handleMessage(decoder *gob.Decoder, encoder *gob.Encoder,
	transmitLock *sync.Mutex) error {
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
	transmitLock.Lock()
	defer transmitLock.Unlock()
	return encoder.Encode(serverMessage)
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
	hashVal, validUntil, err := pathMgr.generator.generate(machine, m.logger)
	if err != nil {
		return fileInfo
	}
	fileInfo.Hash = hashVal
	fileInfo.ValidUntil = validUntil
	m.rwMutex.Lock()
	defer m.rwMutex.Unlock()
	pathMgr.machineHashes[machine.Hostname] = expiringHash{hashVal, validUntil}
	return fileInfo
}
