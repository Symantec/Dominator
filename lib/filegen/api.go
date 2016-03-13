/*
	Package filegen manages the generation of computed files.

	Package filegen may be used to implement a computed file server. It
	registers a FileGenerator server with the lib/srpc package. The application
	may register multiple file generators.

	A generator for the /etc/mdb.json pathname is automatically registered.
*/
package filegen

import (
	"github.com/Symantec/Dominator/lib/hash"
	"github.com/Symantec/Dominator/lib/mdb"
	"github.com/Symantec/Dominator/lib/objectserver/memory"
	proto "github.com/Symantec/Dominator/proto/filegenerator"
	"io"
	"log"
	"sync"
	"time"
)

// FileGenerator is the interface that wraps the Generate method.
//
// Generate computes file data from the provided machine information.
// The logger may be used to log problems.
// It returns the data, a time.Time indicating when the data are valid until
// (a zero time indicates the data are valid forever) and an error.
type FileGenerator interface {
	Generate(machine mdb.Machine, logger *log.Logger) (
		data []byte, validUntil time.Time, err error)
}

type expiringHash struct {
	hash       hash.Hash
	validUntil time.Time
}

type pathManager struct {
	generator hashGenerator
	rwMutex   sync.RWMutex
	// Protected by lock.
	machineHashes map[string]expiringHash
}

type Manager struct {
	rwMutex sync.RWMutex
	// Protected by lock.
	pathManagers map[string]*pathManager
	machineData  map[string]mdb.Machine
	clients      map[<-chan *proto.ServerMessage]chan<- *proto.ServerMessage
	// Not protected by lock.
	objectServer *memory.ObjectServer
	logger       *log.Logger
}

// New creates a new *Manager. Only one should be created per application.
// The logger will be used to log problems.
func New(logger *log.Logger) *Manager {
	return newManager(logger)
}

// GetRegisteredPaths returns a slice of filenames which have generators.
func (m *Manager) GetRegisteredPaths() []string {
	return m.getRegisteredPaths()
}

// RegisterFileForPath registers a source file for a specific pathname. The
// source file is used as the data source. If the source file changes, the data
// are re-read.
func (m *Manager) RegisterFileForPath(pathname string, sourceFile string) {
	m.registerFileForPath(pathname, sourceFile)
}

// RegisterGeneratorForPath registers a FileGenerator for a specific pathname.
// It returns a channel to which notification messages may be sent indicating
// that the data should be regenerated, even if the machine data has not
// changed. If the empty string is sent to the channel, it indicates that data
// should be regenerated for all machines, otherwise it indicates that data
// should be regenerated for a specific machine.
// An internal goroutine reads from the channel, which terminates if the channel
// is closed. The channel should be closed if the data should only be
// regenerated if the machine data changes.
func (m *Manager) RegisterGeneratorForPath(pathname string,
	gen FileGenerator) chan<- string {
	return m.registerDataGeneratorForPath(pathname, gen)
}

// RegisterTemplateFileForPath registers a template file for a specific
// pathname.
// The template file is used to generate the data, modified by the machine data.
// If the template file changes and watchForUpdates is true, the template file
// is re-read and the data are regenerated.
// The template file syntax is defined by the text/template standard package.
func (m *Manager) RegisterTemplateFileForPath(pathname string,
	templateFile string, watchForUpdates bool) error {
	return m.registerTemplateFileForPath(pathname, templateFile,
		watchForUpdates)
}

// WriteHtml will write status information about the Manager to w, with
// appropriate HTML markups.
func (m *Manager) WriteHtml(writer io.Writer) {
	m.writeHtml(writer)
}
