package filegen

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/srpc"
	"log"
	"time"
)

type jsonType struct{}

type rpcType struct {
	manager *Manager
}

func newManager(logger *log.Logger) *Manager {
	manager := new(Manager)
	manager.pathManagers = make(map[string]*pathManager)
	manager.logger = logger
	close(manager.registerGeneratorForPath("/etc/mdb.json", jsonType{}))
	srpc.RegisterName("FileGenerator", &rpcType{manager})
	return manager
}

func (jsonType) Generate(machine mdb.Machine, logger *log.Logger) (
	[]byte, time.Time, error) {
	buffer := new(bytes.Buffer)
	if err := json.WriteWithIndent(buffer, "    ", machine); err != nil {
		return nil, time.Time{}, err
	}
	return buffer.Bytes(), time.Time{}, nil
}
