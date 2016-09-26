package main

import (
	"encoding/gob"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	"github.com/Symantec/Dominator/proto/mdbserver"
	"io"
	"log"
	"sync"
)

type rpcType struct {
	currentMdb *mdb.Mdb
	logger     *log.Logger
	rwMutex    sync.RWMutex
	// Protected by lock.
	updateChannels map[*srpc.Conn]chan<- mdbserver.MdbUpdate
}

func startRpcd(logger *log.Logger) func(old, new *mdb.Mdb) {
	rpcObj := &rpcType{
		logger:         logger,
		updateChannels: make(map[*srpc.Conn]chan<- mdbserver.MdbUpdate),
	}
	srpc.RegisterName("MdbServer", rpcObj)
	return rpcObj.pushUpdateToAll
}

func (t *rpcType) GetMdbUpdates(conn *srpc.Conn) error {
	encoder := gob.NewEncoder(conn)
	updateChannel := make(chan mdbserver.MdbUpdate, 1)
	t.rwMutex.Lock()
	t.updateChannels[conn] = updateChannel
	t.rwMutex.Unlock()
	mdbUpdate := mdbserver.MdbUpdate{MachinesToAdd: t.currentMdb.Machines}
	if err := encoder.Encode(mdbUpdate); err != nil {
		return nil
	}
	if err := conn.Flush(); err != nil {
		return nil
	}
	closeChannel := getCloseNotifier(conn)
	for {
		var err error
		select {
		case mdbUpdate := <-updateChannel:
			if err = encoder.Encode(mdbUpdate); err != nil {
				break
			}
			if err = conn.Flush(); err != nil {
				break
			}
		case err = <-closeChannel:
			break
		}
		if err != nil {
			t.rwMutex.Lock()
			delete(t.updateChannels, conn)
			t.rwMutex.Unlock()
			if err != io.EOF {
				t.logger.Println(err)
				return err
			} else {
				return nil
			}
		}
	}
}

func (t *rpcType) pushUpdateToAll(old, new *mdb.Mdb) {
	t.currentMdb = new
	t.rwMutex.RLock()
	numConnections := len(t.updateChannels)
	t.rwMutex.RUnlock()
	if numConnections < 1 {
		return
	}
	mdbUpdate := mdbserver.MdbUpdate{}
	oldMachines := make(map[string]mdb.Machine, len(old.Machines))
	for _, machine := range old.Machines {
		oldMachines[machine.Hostname] = machine
	}
	for _, newMachine := range new.Machines {
		if oldMachine, ok := oldMachines[newMachine.Hostname]; ok {
			if newMachine != oldMachine {
				mdbUpdate.MachinesToUpdate = append(mdbUpdate.MachinesToUpdate,
					newMachine)
			}
		} else {
			mdbUpdate.MachinesToAdd = append(mdbUpdate.MachinesToAdd,
				newMachine)
		}
	}
	for _, machine := range new.Machines {
		delete(oldMachines, machine.Hostname)
	}
	for name := range oldMachines {
		mdbUpdate.MachinesToDelete = append(mdbUpdate.MachinesToDelete, name)
	}
	t.rwMutex.RLock()
	defer t.rwMutex.RUnlock()
	for _, channel := range t.updateChannels {
		channel <- mdbUpdate
	}
}

func getCloseNotifier(conn *srpc.Conn) <-chan error {
	closeChannel := make(chan error)
	go func() {
		for {
			buf := make([]byte, 1)
			if _, err := conn.Read(buf); err != nil {
				closeChannel <- err
				return
			}
		}
	}()
	return closeChannel
}
