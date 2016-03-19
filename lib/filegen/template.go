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
	objectServer    *memory.ObjectServer
	logger          *log.Logger
	template        *template.Template
	notifierChannel chan<- string
}

func (m *Manager) registerTemplateFileForPath(pathname string,
	templateFile string, watchForUpdates bool) error {
	tgen := &templateGenerator{
		objectServer: m.objectServer,
		logger:       m.logger}
	tgen.notifierChannel = m.registerHashGeneratorForPath(pathname, tgen)
	if watchForUpdates {
		readCloserChannel := fsutil.WatchFile(templateFile, m.logger)
		go tgen.handleReadClosers(readCloserChannel)
	} else {
		file, err := os.Open(templateFile)
		if err != nil {
			return err
		}
		if err := tgen.handleReadCloser(file); err != nil {
			return err
		}
	}
	return nil
}

func (tgen *templateGenerator) generate(machine mdb.Machine,
	logger *log.Logger) (
	hash.Hash, uint64, time.Time, error) {
	if tgen.template == nil {
		return hash.Hash{}, 0, time.Time{}, errors.New("no template data yet")
	}
	buffer := new(bytes.Buffer)
	if err := tgen.template.Execute(buffer, machine); err != nil {
		return hash.Hash{}, 0, time.Time{}, err
	}
	length := uint64(buffer.Len())
	hashVal, _, err := tgen.objectServer.AddObject(buffer, length, nil)
	return hashVal, length, time.Time{}, err
}

func (tgen *templateGenerator) handleReadClosers(
	readCloserChannel <-chan io.ReadCloser) {
	for readCloser := range readCloserChannel {
		if err := tgen.handleReadCloser(readCloser); err != nil {
			tgen.logger.Println(err)
		}
	}
}

func (tgen *templateGenerator) handleReadCloser(
	readCloser io.ReadCloser) error {
	data, err := ioutil.ReadAll(readCloser)
	readCloser.Close()
	if err != nil {
		return err
	}
	tmpl, err := template.New("generatorTemplate").Parse(string(data))
	if err != nil {
		return err
	}
	tgen.template = tmpl
	tgen.notifierChannel <- ""
	return nil
}
