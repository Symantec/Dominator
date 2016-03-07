package filegen

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"log"
	"time"
)

type fileGenerator struct {
	sourceFile string
}

func (m *Manager) registerFileForPath(pathname string, sourceFile string) {
	m.registerGeneratorForPath(pathname, &fileGenerator{sourceFile})
}

func (fgen *fileGenerator) Generate(machine mdb.Machine, logger *log.Logger) (
	[]byte, time.Time, error) {
	panic("not implemented") // TODO(rgooch): Implement.
}
