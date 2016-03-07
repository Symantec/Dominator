/*
	Package filegen manages the generation of computed files.

	Package filegen may be used to implement a computed file server. It
	registers a FileGenerator server with the lib/srpc package. The application
	may register multiple file generators.

	A generator for the /etc/mdb.json pathname is automatically registered.
*/
package filegen

import (
	"github.com/Symantec/Dominator/lib/mdb"
	"io"
	"log"
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

type Manager struct {
	fileGenerators map[string]FileGenerator
	logger         *log.Logger
}

// New creates a new *Manager. Only one should be created per application.
// The logger will be used to log problems.
func New(logger *log.Logger) *Manager {
	return newManager(logger)
}

// RegisterFileForPath registers a source file for a specific pathname. The
// source file is used as the data source.
func (m *Manager) RegisterFileForPath(pathname string, sourceFile string) {
	m.registerFileForPath(pathname, sourceFile)
}

// RegisterGeneratorForPath registers a FileGenerator for a specific pathname.
func (m *Manager) RegisterGeneratorForPath(pathname string, gen FileGenerator) {
	m.registerGeneratorForPath(pathname, gen)
}

// WriteHtml will write status information about the Manager to w, with
// appropriate HTML markups.
func (m *Manager) WriteHtml(writer io.Writer) {
	m.writeHtml(writer)
}
