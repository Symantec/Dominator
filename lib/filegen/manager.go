package filegen

import (
	"github.com/Symantec/Dominator/lib/log"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	"github.com/Symantec/Dominator/lib/srpc"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
)

type rpcType struct {
	manager *Manager
}

func newManager(logger log.Logger) *Manager {
	m := new(Manager)
	m.pathManagers = make(map[string]*pathManager)
	m.machineData = make(map[string]mdb.Machine)
	m.clients = make(
		map[<-chan *proto.ServerMessage]chan<- *proto.ServerMessage)
	m.objectServer = memory.NewObjectServer()
	m.logger = logger
	m.registerMdbGeneratorForPath("/etc/mdb.json")
	srpc.RegisterName("FileGenerator", &rpcType{m})
	return m
}
