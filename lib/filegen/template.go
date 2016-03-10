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
	"os"
	"text/template"
	"time"
)

type templateGenerator struct {
	objectServer *memory.ObjectServer
	logger       *log.Logger
	template     *template.Template
}

func (m *Manager) registerTemplateFileForPath(pathname string,
	templateFile string, watchForUpdates bool) error {
	tgen := &templateGenerator{
		objectServer: m.objectServer,
		logger:       m.logger}
	if watchForUpdates {
		readerChannel := fsutil.WatchFile(templateFile, m.logger)
		go tgen.handleReaders(readerChannel)
	} else {
		file, err := os.Open(pathname)
		if err != nil {
			return err
		}
		if err := tgen.handleReader(file); err != nil {
			return err
		}
	}
	m.registerHashGeneratorForPath(pathname, tgen)
	return nil
}

func (tgen *templateGenerator) generate(machine mdb.Machine, logger *log.Logger) (
	hash.Hash, time.Time, error) {
	if tgen.template == nil {
		return hash.Hash{}, time.Time{}, errors.New("no template data yet")
	}
	buffer := new(bytes.Buffer)
	if err := tgen.template.Execute(buffer, machine); err != nil {
		return hash.Hash{}, time.Time{}, err
	}
	hashVal, _, err := tgen.objectServer.AddObject(buffer, uint64(buffer.Len()),
		nil)
	return hashVal, time.Time{}, err
}

func (tgen *templateGenerator) handleReaders(readerChannel <-chan io.Reader) {
	for reader := range readerChannel {
		if err := tgen.handleReader(reader); err != nil {
			tgen.logger.Println(err)
		}
	}
}

func (tgen *templateGenerator) handleReader(reader io.Reader) error {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	tmpl, err := template.New("generatorTemplate").Parse(string(data))
	if err != nil {
		return err
	}
	tgen.template = tmpl
	return nil
}
