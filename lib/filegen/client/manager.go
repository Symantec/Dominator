package client

import (
	"bytes"
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/objectserver"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io"
	"log"
	"time"
)

func newManager(objSrv objectserver.ObjectServer, logger *log.Logger) *Manager {
	sourceConnectChannel := make(chan string)
	m := &Manager{
		sourceMap:            make(map[string]*sourceType),
		objectServer:         objSrv,
		machineMap:           make(map[string]*machineType),
		addMachineChannel:    make(chan *machineType),
		removeMachineChannel: make(chan string),
		updateMachineChannel: make(chan *machineType),
		serverMessageChannel: make(chan *serverMessageType),
		sourceConnectChannel: sourceConnectChannel,
		objectWaiters:        make(map[hash.Hash][]chan<- hash.Hash),
		logger:               logger}
	go m.manage(sourceConnectChannel)
	return m
}

func (m *Manager) manage(sourceConnectChannel <-chan string) {
	for {
		select {
		case machine := <-m.addMachineChannel:
			m.addMachine(machine)
		case hostname := <-m.removeMachineChannel:
			m.removeMachine(hostname)
		case machine := <-m.updateMachineChannel:
			m.updateMachine(machine)
		case serverMessage := <-m.serverMessageChannel:
			m.processMessage(serverMessage)
		case sourceName := <-sourceConnectChannel:
			m.processSourceConnect(sourceName)
		}
	}
}

func (m *Manager) processMessage(serverMessage *serverMessageType) {
	if msg := serverMessage.serverMessage.GetObjectResponse; msg != nil {
		if _, _, err := m.objectServer.AddObject(
			bytes.NewReader(msg.Data), 0, &msg.Hash); err != nil {
			m.logger.Println(err)
		} else {
			if waiters, ok := m.objectWaiters[msg.Hash]; ok {
				for _, channel := range waiters {
					channel <- msg.Hash
				}
				delete(m.objectWaiters, msg.Hash)
			}
		}
	}
	if msg := serverMessage.serverMessage.YieldResponse; msg != nil {
		if machine, ok := m.machineMap[msg.Hostname]; ok {
			m.handleYieldResponse(machine, msg.Files)
		} // else machine no longer known. Drop the message.
	}
}

func (m *Manager) processSourceConnect(sourceName string) {
	source := m.sourceMap[sourceName]
	for _, machine := range m.machineMap {
		if pathnames, ok := machine.sourceToPaths[sourceName]; ok {
			request := &proto.ClientRequest{
				YieldRequest: &proto.YieldRequest{machine.machine, pathnames}}
			source.sendChannel <- request
		}
	}
}

// Returns true if the source was already set up.
func (m *Manager) getSource(sourceName string) (*sourceType, bool) {
	source, ok := m.sourceMap[sourceName]
	if ok {
		return source, true
	}
	source = new(sourceType)
	sendChannel := make(chan *proto.ClientRequest, 4096)
	source.sendChannel = sendChannel
	m.sourceMap[sourceName] = source
	go manageSource(sourceName, m.sourceConnectChannel, sendChannel,
		m.serverMessageChannel, m.logger)
	return source, false
}

func manageSource(sourceName string, sourceConnectChannel chan<- string,
	clientRequestChannel <-chan *proto.ClientRequest,
	serverMessageChannel chan<- *serverMessageType, logger *log.Logger) {
	closeNotifyChannel := make(chan struct{})
	initialRetryTimeout := time.Millisecond * 100
	retryTimeout := initialRetryTimeout
	for ; ; time.Sleep(retryTimeout) {
		if retryTimeout < time.Minute {
			retryTimeout *= 2
		}
		client, err := srpc.DialHTTP("tcp", sourceName, time.Second*15)
		if err != nil {
			logger.Printf("error connecting to: %s: %s\n", sourceName, err)
			continue
		}
		conn, err := client.Call("FileGenerator.Connect")
		if err != nil {
			client.Close()
			logger.Println(err)
			continue
		}
		retryTimeout = initialRetryTimeout
		// The server keeps the same encoder/decoder pair over the lifetime of
		// the connection, so we must do the same.
		go handleServerMessages(sourceName, gob.NewDecoder(conn),
			serverMessageChannel, closeNotifyChannel, logger)
		sourceConnectChannel <- sourceName
		sendClientRequests(conn, clientRequestChannel, closeNotifyChannel,
			logger)
		conn.Close()
		client.Close()
	}
}

func sendClientRequests(conn *srpc.Conn,
	clientRequestChannel <-chan *proto.ClientRequest,
	closeNotifyChannel <-chan struct{}, logger *log.Logger) {
	encoder := gob.NewEncoder(conn)
	for {
		select {
		case clientRequest := <-clientRequestChannel:
			if err := encoder.Encode(clientRequest); err != nil {
				logger.Printf("error encoding client request: %s\n", err)
				return
			}
			if len(clientRequestChannel) < 1 {
				if err := conn.Flush(); err != nil {
					logger.Printf("error flushing: %s\n", err)
					return
				}
			}
		case <-closeNotifyChannel:
			return
		}
	}
}

func handleServerMessages(sourceName string, decoder *gob.Decoder,
	serverMessageChannel chan<- *serverMessageType,
	closeNotifyChannel chan<- struct{}, logger *log.Logger) {
	for {
		var message proto.ServerMessage
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				logger.Printf("connection to source: %s closed\n", sourceName)
			} else {
				logger.Println(err)
			}
			closeNotifyChannel <- struct{}{}
			return
		}
		serverMessageChannel <- &serverMessageType{sourceName, message}
	}
}
