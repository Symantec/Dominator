/*
	Package logbuf provides a circular buffer for writing logs to.

	Package logbuf provides an io.Writer which can be passed to the log.New
	function to serve as a destination for logs. Logs can be viewed via a HTTP
	interface and may also be directed to the standard error output.
*/
package logbuf

import (
	"bufio"
	"container/ring"
	"io"
	"os"
	"sync"
)

// LogBuffer is a circular buffer suitable for holding logs. It satisfies the
// io.Writer interface. It is usually passed to the log.New function.
type LogBuffer struct {
	rwMutex       sync.RWMutex
	buffer        *ring.Ring // Always points to next insert position.
	logDir        string
	file          *os.File
	writer        *bufio.Writer
	usage         uint64
	quota         uint64
	writeNotifier chan<- struct{}
}

// New returns a *LogBuffer with the specified number of lines of buffer.
// Only one should be created per application.
func New(length uint) *LogBuffer {
	return newLogBuffer(length, *logDir, uint64(*logQuota)<<20)
}

// Write will write len(p) bytes from p to the log buffer. It always returns
// len(p), nil.
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	return lb.write(p)
}

// Dump will write the contents of the log buffer to w, with a prefix and postfix
// string written before and after each line.
func (lb *LogBuffer) Dump(writer io.Writer, prefix, postfix string) error {
	return lb.dump(writer, prefix, postfix)
}

// WriteHtml will write the contents of the log buffer to w, with appropriate
// HTML markups.
func (lb *LogBuffer) WriteHtml(writer io.Writer) {
	lb.writeHtml(writer)
}
