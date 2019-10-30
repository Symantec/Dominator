/*
	Package logbuf provides a circular buffer for writing logs to.

	Package logbuf provides an io.Writer which can be passed to the log.New
	function to serve as a destination for logs. Logs can be viewed via a HTTP
	interface and may also be directed to the standard error output.
*/
package logbuf

import (
	"container/ring"
	"flag"
	"io"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/Cloud-Foundations/Dominator/lib/bufwriter"
	"github.com/Cloud-Foundations/Dominator/lib/flagutil"
)

var (
	stdOptions = Options{
		HttpServeMux: http.DefaultServeMux,
		MaxFileSize:  10 << 20,
		Quota:        100 << 20,
	}
	kSoleLogBuffer *LogBuffer
	kOnce          sync.Once
)

// LogBuffer is a circular buffer suitable for holding logs. It satisfies the
// io.Writer interface. It is usually passed to the log.New function.
type LogBuffer struct {
	options       Options
	rwMutex       sync.RWMutex
	buffer        *ring.Ring // Always points to next insert position.
	file          *os.File
	writer        *bufwriter.Writer
	fileSize      flagutil.Size
	usage         flagutil.Size
	writeNotifier chan<- struct{}
	panicLogfile  *string // Name of last invocation logfile if it has a panic.
}

type Options struct {
	AlsoLogToStderr bool
	Directory       string
	HttpServeMux    *http.ServeMux
	IdleMarkTimeout time.Duration
	MaxBufferLines  uint          // Minimum: 100.
	MaxFileSize     flagutil.Size // Minimum: 16 KiB
	Quota           flagutil.Size // Minimum: 64 KiB.
	RedirectStderr  bool          // Only one LogBuffer should set this.
}

// UseFlagSet instructs this package to read its command-line flags from the
// given flag set instead of from the command line. Caller must pass the
// flag set to this method before calling Parse on it.
func UseFlagSet(set *flag.FlagSet) {
	set.BoolVar(&stdOptions.AlsoLogToStderr, "alsoLogToStderr", false,
		"If true, also write logs to stderr")
	set.DurationVar(&stdOptions.IdleMarkTimeout, "idleMarkTimeout", 0,
		"time after last log before a 'MARK' message is written to logfile")
	set.UintVar(&stdOptions.MaxBufferLines, "logbufLines", 1024,
		"Number of lines to store in the log buffer")
	set.StringVar(&stdOptions.Directory, "logDir", path.Join("/var/log",
		path.Base(os.Args[0])),
		"Directory to write log data to. If empty, no logs are written")
	set.Var(&stdOptions.MaxFileSize, "logFileMaxSize",
		"Maximum size for a log file. If exceeded, new file is created")
	set.Var(&stdOptions.Quota, "logQuota",
		"Log quota. If exceeded, old logs are deleted")
}

// GetStandardOptions will return the standard options.
// Only one *LogBuffer should be created per application with these options.
// The following command-line flags are registered and used:
//  -alsoLogToStderr: If true, also write logs to stderr
//  -logbufLines:     Number of lines to store in the log buffer
//  -logDir:          Directory to write log data to. If empty, no logs are
//                    written
//  -logFileMaxSize:  Maximum size for each log file. If exceeded, the logfile
//                    is closed and a new one opened.
//                    If zero, the limit will be 16 KiB
//  -logQuota:        Log quota. If exceeded, old logs are deleted.
//                    If zero, the quota will be 64 KiB
func GetStandardOptions() Options { return stdOptions }

// New returns a new *LogBuffer with the standard options. Note that
// RedirectStderr will be set to true if AlsoLogToStderr is false.
// Only one should be created per application.
func New() *LogBuffer {
	options := stdOptions
	if !options.AlsoLogToStderr {
		options.RedirectStderr = true
	}
	return newLogBuffer(options)
}

// NewWithOptions will create a new *LogBuffer with the specified options.
// Each *LogBuffer must use a different Directory and HttpServeMux.
func NewWithOptions(options Options) *LogBuffer {
	return newLogBuffer(options)
}

// Get works like New except that successive calls to Get return the same
// instance.
func Get() *LogBuffer {
	kOnce.Do(func() {
		kSoleLogBuffer = New()
	})
	return kSoleLogBuffer

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

// WriteHtml will write the contents of the log buffer to writer, with
// appropriate HTML markups.
func (lb *LogBuffer) WriteHtml(writer io.Writer) {
	lb.writeHtml(writer)
}
