package serverlogger

import (
	"flag"
	"fmt"
	"github.com/Symantec/Dominator/lib/log/debuglogger"
	"github.com/Symantec/Dominator/lib/logbuf"
	"io"
	"os"
)

var (
	initialLogDebugLevel = flag.Int("initialLogDebugLevel", -1,
		"initial debug log level")
)

type Logger struct {
	*debuglogger.Logger
	circularBuffer *logbuf.LogBuffer
}

// New will create a Logger which has an internal log buffer (see the
// lib/logbuf package). It implements the log.DebugLogger interface.
// By default, the max debug level is -1, meaning all debug logs are dropped
// (ignored).
// The name of the new logger is given by name. This name is used to remotely
// identify the logger for RPC methods such as Logger.SetDebugLevel. The first
// or primary logger should be created with name "" (the empty string).
func New(name string) *Logger {
	return newLogger(name)
}

func (l *Logger) Fatal(v ...interface{}) {
	msg := fmt.Sprint(v...)
	l.Print(msg)
	l.circularBuffer.Flush()
	os.Exit(1)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Fatal(fmt.Sprintf(format, v...))
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.Fatal(fmt.Sprintln(v...))
}

// Flush flushes the open log file (if one is open). This should only be called
// just prior to process termination. The log file is automatically flushed
// after short periods of inactivity.
func (l *Logger) Flush() error {
	return l.circularBuffer.Flush()
}

// WriteHtml will write the contents of the internal log buffer to writer, with
// appropriate HTML markups.
func (l *Logger) WriteHtml(writer io.Writer) {
	l.circularBuffer.WriteHtml(writer)
}
