package filegen

import (
	"errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	"io"
	"log"
	"time"
)

type fileGenerator struct {
	objectServer    *memory.ObjectServer
	logger          *log.Logger
	hash            *hash.Hash
	notifierChannel chan<- string
}

func (m *Manager) registerFileForPath(pathname string, sourceFile string) {
	readerChannel := fsutil.WatchFile(sourceFile, m.logger)
	fgen := &fileGenerator{
		objectServer: m.objectServer,
		logger:       m.logger}
	fgen.notifierChannel = m.registerHashGeneratorForPath(pathname, fgen)
	go fgen.handleReaders(readerChannel)

}

func (fgen *fileGenerator) generate(machine mdb.Machine, logger *log.Logger) (
	hash.Hash, time.Time, error) {
	if fgen.hash == nil {
		return hash.Hash{}, time.Time{}, errors.New("no hash yet")
	}
	return *fgen.hash, time.Time{}, nil
}

func (fgen *fileGenerator) handleReaders(readerChannel <-chan io.Reader) {
	for reader := range readerChannel {
		hashVal, _, err := fgen.objectServer.AddObject(reader, 0, nil)
		if err != nil {
			fgen.logger.Println(err)
			continue
		}
		fgen.hash = &hashVal
		fgen.notifierChannel <- ""
	}
}
