package filegen

import (
	"bytes"
	"errors"
	"github.com/Symantec/Dominator/lib/fsutil"
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	"io"
	"io/ioutil"
	"log"
	"time"
)

type fileGenerator struct {
	objectServer *memory.ObjectServer
	logger       *log.Logger
	hash         *hash.Hash
}

func (m *Manager) registerFileForPath(pathname string, sourceFile string) {
	readerChannel := fsutil.WatchFile(sourceFile, m.logger)
	fgen := &fileGenerator{
		objectServer: m.objectServer,
		logger:       m.logger}
	go fgen.handleReaders(readerChannel)
	m.registerGeneratorForPath(pathname, fgen)
}

func (fgen *fileGenerator) Generate(machine mdb.Machine, logger *log.Logger) (
	[]byte, time.Time, error) {
	if fgen.hash == nil {
		return nil, time.Time{}, errors.New("no hash yet")
	}
	_, reader, err := fgen.objectServer.GetObject(*fgen.hash)
	if err != nil {
		panic("no object for hash")
	}
	data, err := ioutil.ReadAll(reader)
	reader.Close()
	if err != nil {
		panic(err)
	}
	return data, time.Time{}, nil
}

func (fgen *fileGenerator) handleReaders(readerChannel <-chan io.Reader) {
	for reader := range readerChannel {
		data, err := ioutil.ReadAll(reader)
		if err != nil {
			fgen.logger.Println(err)
			continue
		}
		hashVal, _, err := fgen.objectServer.AddObject(bytes.NewReader(data),
			uint64(len(data)), nil)
		if err != nil {
			fgen.logger.Println(err)
			continue
		}
		fgen.hash = &hashVal
	}
}
