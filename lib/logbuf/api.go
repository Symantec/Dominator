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
	"flag"
	"io"
	"os"
	"path"
	"sync"
)

var (
	alsoLogToStderr = flag.Bool("alsoLogToStderr", false,
		"If true, also write logs to stderr")
	idleMarkTimeout = flag.Duration("idleMarkTimeout", 0,
		"time after last log before a 'MARK' message is written to logfile")
	logbufLines = flag.Uint("logbufLines", 1024,
		"Number of lines to store in the log buffer")
	logDir = flag.String("logDir", path.Join("/var/log", path.Base(os.Args[0])),
		"Directory to write log data to. If empty, no logs are written")
	logQuota = flag.Uint("logQuota", 10,
		"Log quota in MiB. If exceeded, old logs are deleted")
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
	panicLogfile  *string // Name of last invocation logfile if it has a panic.
}

// New returns a *LogBuffer with the specified number of lines of buffer.
// Only one should be created per application.
// The behaviour of the LogBuffer is controlled by the following command-line
// flags (registered with the standard flag pacakge):
//  -alsoLogToStderr: If true, also write logs to stderr
//  -logbufLines:     Number of lines to store in the log buffer
//  -logDir:          Directory to write log data to. If empty, no logs are
//                    written
//  -logQuota:        Log quota in MiB. If exceeded, old logs are deleted.
//                    If zero, the quota will be 16 KiB
//  -minLogLevel      Minimum log level. A log message must have at least
//                    this level to be recorded. Must be one of the predefined
//                    levels out of logbuf/level package such as
//                    TRACE, DEBUG, INFO, WARN, or ERR.
//                    If it isn't one of these values, the default is record
//                    all logs. See logbuf/level package for more details
//                    on assigning a level to a log message.
func New() *LogBuffer {
	quota := uint64(*logQuota) << 20
	if quota < 16384 {
		quota = 16384
	}
	return newLogBuffer(*logbufLines, *logDir, quota)
}

// Dump will write the contents of the log buffer to w, with a prefix and
// postfix string written before and after each line. If recentFirst is true,
// the most recently written contents are dumped first.
func (lb *LogBuffer) Dump(writer io.Writer, prefix, postfix string,
	recentFirst bool) error {
	return lb.dump(writer, prefix, postfix, recentFirst)
}

// Flush flushes the open log file (if one is open). This should only be called
// just prior to process termination. The log file is automatically flushed
// after short periods of inactivity.
func (lb *LogBuffer) Flush() error {
	return lb.flush()
}

// Write will write len(p) bytes from p to the log buffer. It always returns
// len(p), nil.
func (lb *LogBuffer) Write(p []byte) (n int, err error) {
	return lb.write(p)
}

// WriteHtml will write the contents of the log buffer to w, with appropriate
// HTML markups.
func (lb *LogBuffer) WriteHtml(writer io.Writer) {
	lb.writeHtml(writer)
}
