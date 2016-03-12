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
	m := &Manager{
		sourceMap:            make(map[string]*sourceType),
		objectServer:         objSrv,
		machineMap:           make(map[string]*machineType),
		addMachineChannel:    make(chan *machineType),
		removeMachineChannel: make(chan string),
		updateMachineChannel: make(chan *machineType),
		serverMessageChannel: make(chan *serverMessageType),
		objectWaiters:        make(map[hash.Hash][]chan<- hash.Hash),
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
		case serverMessage := <-m.serverMessageChannel:
			m.processMessage(serverMessage)
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

func sendClientRequests(source string, channel <-chan proto.ClientRequest,
	serverMessageChannel chan<- *serverMessageType, logger *log.Logger) {
	var client *srpc.Client
	var conn *srpc.Conn
	// The server keeps the same encoder/decoder pair over the lifetime of the
	// connection, so we must do the same.
	var encoder *gob.Encoder
	for request := range channel {
		client, conn, encoder = sendClientRequest(source, client, conn, encoder,
			request, serverMessageChannel, logger)
		if len(channel) < 1 {
			conn.Flush()
		}
	}
}

func sendClientRequest(source string, client *srpc.Client, conn *srpc.Conn,
	encoder *gob.Encoder, request proto.ClientRequest,
	serverMessageChannel chan<- *serverMessageType, logger *log.Logger) (
	*srpc.Client, *srpc.Conn, *gob.Encoder) {
	for {
		var err error
		if conn == nil {
			client, err = srpc.DialHTTP("tcp", source, time.Second*15)
			if err != nil {
				logger.Println(err)
				time.Sleep(time.Second * 15)
				continue
			}
			conn, err = client.Call("FileGenerator.Connect")
			if err != nil {
				client.Close()
				client = nil
				logger.Println(err)
				time.Sleep(time.Second * 15)
				continue
			}
			encoder = gob.NewEncoder(conn)
			go receiveServerMessages(source, conn, serverMessageChannel, logger)
		}
		if err := encoder.Encode(request); err != nil {
			conn.Close()
			conn = nil
			client.Close()
			client = nil
			logger.Println(err)
			time.Sleep(time.Second * 15)
			continue
		}
		return client, conn, encoder
	}
}

func receiveServerMessages(source string, conn *srpc.Conn,
	serverMessageChannel chan<- *serverMessageType, logger *log.Logger) {
	decoder := gob.NewDecoder(conn)
	for {
		var message proto.ServerMessage
		if err := decoder.Decode(&message); err != nil {
			if err != io.EOF {
				logger.Println(err)
			}
			return
		}
		serverMessageChannel <- &serverMessageType{source, message}
	}
}
