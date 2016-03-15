package filegen

import (
	"bytes"
	"github.com/Symantec/Dominator/lib/json"
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
	"time"
)

type jsonType struct{}

func (m *Manager) registerMdbGeneratorForPath(pathname string) {
	close(m.registerDataGeneratorForPath(pathname, jsonType{}))
}

func (jsonType) Generate(machine mdb.Machine, logger *log.Logger) (
	[]byte, time.Time, error) {
	buffer := new(bytes.Buffer)
	if err := json.WriteWithIndent(buffer, "    ", machine); err != nil {
		return nil, time.Time{}, err
	}
	buffer.WriteString("\n")
	return buffer.Bytes(), time.Time{}, nil
}
