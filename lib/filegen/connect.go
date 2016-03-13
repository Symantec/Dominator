package filegen

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io"
	"io/ioutil"
	"time"
)

func (t *rpcType) Connect(conn *srpc.Conn) error {
	return t.manager.connect(conn) // Long-lived.
}

func (m *Manager) connect(conn *srpc.Conn) error {
	defer conn.Flush()
	clientChannel := make(chan *proto.ServerMessage, 4096)
	m.rwMutex.Lock()
	m.clients[clientChannel] = clientChannel
	m.rwMutex.Unlock()
	defer func() {
		m.rwMutex.Lock()
		delete(m.clients, clientChannel)
		m.rwMutex.Unlock()
	}()
	closeNotifyChannel := make(chan struct{})
	// The client must keep the same encoder/decoder pair over the lifetime
	// of the connection.
	go m.handleClientRequests(gob.NewDecoder(conn), clientChannel,
		closeNotifyChannel)
	encoder := gob.NewEncoder(conn)
	for {
		select {
		case serverMessage := <-clientChannel:
			if err := encoder.Encode(serverMessage); err != nil {
				m.logger.Printf("error encoding ServerMessage: %s\n", err)
				return err
			}
			if len(clientChannel) < 1 {
				if err := conn.Flush(); err != nil {
					m.logger.Printf("error flushing: %s\n", err)
					return err
				}
			}
		case <-closeNotifyChannel:
			return nil
		}
	}
}

func (m *Manager) handleClientRequests(decoder *gob.Decoder,
	messageChan chan<- *proto.ServerMessage,
	closeNotifyChannel chan<- struct{}) {
	for {
		if err := m.handleRequest(decoder, messageChan); err != nil {
			if err != io.EOF {
				m.logger.Println(err)
			}
			closeNotifyChannel <- struct{}{}
		}
	}
}

func (m *Manager) handleRequest(decoder *gob.Decoder,
	messageChan chan<- *proto.ServerMessage) error {
	var request proto.ClientRequest
	if err := decoder.Decode(&request); err != nil {
		if err == io.EOF {
			return err
		}
		return errors.New("error decoding ClientRequest: " + err.Error())
	}
	serverMessage := new(proto.ServerMessage)
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
			return err
		} else {
			data, _ := ioutil.ReadAll(reader)
			serverMessage.GetObjectResponse = &proto.GetObjectResponse{
				Hash: request.Hash,
				Data: data}
			reader.Close()
		}
	}
	messageChan <- serverMessage
	return nil
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
