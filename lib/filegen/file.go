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
	length          uint64
	notifierChannel chan<- string
}

func (m *Manager) registerFileForPath(pathname string, sourceFile string) {
	readCloserChannel := fsutil.WatchFile(sourceFile, m.logger)
	fgen := &fileGenerator{
		objectServer: m.objectServer,
		logger:       m.logger}
	fgen.notifierChannel = m.registerHashGeneratorForPath(pathname, fgen)
	go fgen.handleReaders(readCloserChannel)

}

func (fgen *fileGenerator) generate(machine mdb.Machine, logger *log.Logger) (
	hash.Hash, uint64, time.Time, error) {
	if fgen.hash == nil {
		return hash.Hash{}, 0, time.Time{}, errors.New("no hash yet")
	}
	return *fgen.hash, fgen.length, time.Time{}, nil
}

func (fgen *fileGenerator) handleReaders(
	readCloserChannel <-chan io.ReadCloser) {
	for readCloser := range readCloserChannel {
		hashVal, _, err := fgen.objectServer.AddObject(readCloser, 0, nil)
		readCloser.Close()
		if err != nil {
			fgen.logger.Println(err)
			continue
		}
		fgen.hash = &hashVal
		hashes := make([]hash.Hash, 1)
		hashes[0] = hashVal
		lengths, err := fgen.objectServer.CheckObjects(hashes)
		if err != nil {
			fgen.logger.Println(err)
			continue
		}
		fgen.length = lengths[0]
		fgen.notifierChannel <- ""
	}
}
