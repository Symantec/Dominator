package filegen

import (
	"encoding/gob"
	"errors"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io"
	"io/ioutil"
	"log"
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
		close(clientChannel)
	}()
	// The client must keep the same encoder/decoder pair over the lifetime
	// of the connection.
	decoder := gob.NewDecoder(conn)
	go handleTransmits(conn, clientChannel, m.logger)
	for {
		if err := m.handleMessage(decoder, clientChannel); err != nil {
			if err == io.EOF {
				return nil
			}
			return errors.New("error handling message: " + err.Error())
		}
	}
}

func handleTransmits(conn *srpc.Conn, messageChan <-chan *proto.ServerMessage,
	logger *log.Logger) {
	encoder := gob.NewEncoder(conn)
	for message := range messageChan {
		if err := encoder.Encode(message); err != nil {
			if err == io.EOF {
				// Drain and terminate.
				for range messageChan {
				}
				return
			}
			logger.Println(err)
		}
		if len(messageChan) < 1 {
			conn.Flush()
		}
	}
}

func (m *Manager) handleMessage(decoder *gob.Decoder,
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
