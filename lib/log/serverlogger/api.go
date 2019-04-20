package serverlogger

import (
	"flag"
	"fmt"
	"io"
	"log"
	"regexp"
	"sync"

	"github.com/Symantec/Dominator/lib/logbuf"
	"github.com/Symantec/Dominator/lib/srpc"
)

var (
	initialLogDebugLevel = flag.Int("initialLogDebugLevel", -1,
		"initial debug log level")
	logSubseconds = flag.Bool("logSubseconds", false,
		"if true, datestamps will have subsecond resolution")
)

type Logger struct {
	accessChecker  func(authInfo *srpc.AuthInformation) bool
	circularBuffer *logbuf.LogBuffer
	flags          int
	level          int16
	maxLevel       int16
	mutex          sync.Mutex // Lock everything below.
	streamers      map[*streamerType]struct{}
}

type streamerType struct {
	debugLevel   int16
	excludeRegex *regexp.Regexp // nil: nothing excluded. Processed after incl.
	includeRegex *regexp.Regexp // nil: everything included.
	output       chan<- []byte
}

// New will create a Logger which has an internal log buffer (see the
// lib/logbuf package). It implements the log.DebugLogger interface.
// By default, the max debug level is -1, meaning all debug logs are dropped
// (ignored).
// The name of the new logger is given by name. This name is used to remotely
// identify the logger for SRPC methods such as Logger.SetDebugLevel. The first
// or primary logger should be created with name "" (the empty string).
func New(name string) *Logger {
	flags := log.LstdFlags
	if *logSubseconds {
		flags |= log.Lmicroseconds
	}
	return newLogger(name, logbuf.GetStandardOptions(), flags)
}

// NewWithFlags will create a Logger which has an internal log buffer (see the
// lib/logbuf package). It implements the log.DebugLogger interface.
// By default, the max debug level is -1, meaning all debug logs are dropped
// (ignored).
// The name of the new logger is given by name. This name is used to remotely
// identify the logger for RPC methods such as Logger.SetDebugLevel. The first
// or primary logger should be created with name "" (the empty string).
func NewWithFlags(name string, flags int) *Logger {
	return newLogger(name, logbuf.GetStandardOptions(), flags)
}

// NewWithOptions will create a Logger which has an internal log buffer (see the
// lib/logbuf package). It implements the log.DebugLogger interface.
// By default, the max debug level is -1, meaning all debug logs are dropped
// (ignored).
// The name of the new logger is given by name. This name is used to remotely
// identify the logger for RPC methods such as Logger.SetDebugLevel. The first
// or primary logger should be created with name "" (the empty string).
func NewWithOptions(name string, options logbuf.Options, flags int) *Logger {
	return newLogger(name, options, flags)
}

// Debug will call the Print method if level is less than or equal to the max
// debug level for the Logger.
func (l *Logger) Debug(level uint8, v ...interface{}) {
	l.debug(int16(level), v...)
}

// Debugf will call the Printf method if level is less than or equal to the max
// debug level for the Logger.
func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	l.debugf(int16(level), format, v...)
}

// Debugln will call the Println method if level is less than or equal to the
// max debug level for the Logger.
func (l *Logger) Debugln(level uint8, v ...interface{}) {
	l.debugln(int16(level), v...)
}

// GetLevel gets the current maximum debug level.
func (l *Logger) GetLevel() int16 {
	return l.level
}

// Fatal is equivalent to Print() followed by a call to os.Exit(1).
func (l *Logger) Fatal(v ...interface{}) {
	l.fatals(fmt.Sprint(v...))
}

// Fatalf is equivalent to Printf() followed by a call to os.Exit(1).
func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.fatals(fmt.Sprintf(format, v...))
}

// Fatalln is equivalent to Println() followed by a call to os.Exit(1).
func (l *Logger) Fatalln(v ...interface{}) {
	l.fatals(fmt.Sprintln(v...))
}

// Flush flushes the open log file (if one is open). This should only be called
// just prior to process termination. The log file is automatically flushed
// after short periods of inactivity.
func (l *Logger) Flush() error {
	return l.circularBuffer.Flush()
}

// Panic is equivalent to Print() followed by a call to panic().
func (l *Logger) Panic(v ...interface{}) {
	l.panics(fmt.Sprint(v...))
}

// Panicf is equivalent to Printf() followed by a call to panic().
func (l *Logger) Panicf(format string, v ...interface{}) {
	l.panics(fmt.Sprintf(format, v...))
}

// Panicln is equivalent to Println() followed by a call to panic().
func (l *Logger) Panicln(v ...interface{}) {
	l.panics(fmt.Sprintln(v...))
}

// Print prints to the logger. Arguments are handled in the manner of fmt.Print.
func (l *Logger) Print(v ...interface{}) {
	l.prints(fmt.Sprint(v...))
}

// Printf prints to the logger. Arguments are handled in the manner of
// fmt.Printf.
func (l *Logger) Printf(format string, v ...interface{}) {
	l.prints(fmt.Sprintf(format, v...))
}

// Println prints to the logger. Arguments are handled in the manner of
// fmt.Println.
func (l *Logger) Println(v ...interface{}) {
	l.prints(fmt.Sprintln(v...))
}

// SetAccessChecker sets the function that is called when SRPC methods are
// called for the Logger. This allows the application to control which users or
// groups are permitted to remotely control the Logger.
func (l *Logger) SetAccessChecker(
	accessChecker func(authInfo *srpc.AuthInformation) bool) {
	l.accessChecker = accessChecker
}

// SetLevel sets the maximum debug level. A negative level will cause all debug
// messages to be dropped.
func (l *Logger) SetLevel(maxLevel int16) {
	l.setLevel(maxLevel)
}

// WriteHtml will write the contents of the internal log buffer to writer, with
// appropriate HTML markups.
func (l *Logger) WriteHtml(writer io.Writer) {
	l.circularBuffer.WriteHtml(writer)
}
