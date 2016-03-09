package filegen

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	"github.com/Symantec/Dominator/lib/srpc"
	"log"
	"time"
)

type jsonType struct{}

type rpcType struct {
	manager *Manager
}

func newManager(logger *log.Logger) *Manager {
	m := new(Manager)
	m.pathManagers = make(map[string]*pathManager)
	m.machineData = make(map[string]mdb.Machine)
	m.notifiers = make(map[<-chan notificationData]chan<- notificationData)
	m.objectServer = memory.NewObjectServer()
	m.logger = logger
	close(m.registerGeneratorForPath("/etc/mdb.json", jsonType{}))
	srpc.RegisterName("FileGenerator", &rpcType{m})
	return m
}

func (jsonType) Generate(machine mdb.Machine, logger *log.Logger) (
	[]byte, time.Time, error) {
	buffer := new(bytes.Buffer)
	if err := json.WriteWithIndent(buffer, "    ", machine); err != nil {
		return nil, time.Time{}, err
	}
	return buffer.Bytes(), time.Time{}, nil
}
