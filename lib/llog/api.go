package llog

import (
	"io"
)

// Logger instances log messages to a particular writer. A Logger instance is
// safe to use with multiple goroutines. Moreover, multiple instances logging
// to the same writer synchronise their writes.
type Logger struct {
	// TODO
}

// New creates a new logger. New uses --logLevel command-line flag to
// decide what debug messages to log. If the --logLevel command-line flag
// is not present, the returned logger will not log any debug messages.
// The Returned logger will always log the other types of messages:
// Fatal, Err, Warn, Info
//
// --logLevel=2 means log debug messages with levels 0, 1, and 2. --logLevel=0
// means log debug messages with level 0 only. --logLevel=images:1 means
// log debug messages with levels 0 and 1 for the "images" subsystem only.
// --logLevel may be used more than once on a command line to specify
// different log levels for different subsystems.
// For example --logLevel=0 --logLevel=images:2 --logLevel=files:3 logs
// debug messages up to level 3 for the files subsystem, up to level 2 for
// the images subsystem and level 0 only for all the other subsystems.
//
// writer is where the returned logger writes its messages. If the writer
// implementation has a no-arg Flush() method that returns an error,
// the returned logger calls it after logging a fatal or panic message
// just before terminating the process. subSystemName is the name of the
// subsystem.
func New(writer io.Writer, subSystemName string) *Logger {
	// TODO
	return nil
}

// Fatal logs an Err message, flushes the writer, and terminates the process.
func (l *Logger) Fatal(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Fatalf(format string, v ...interface{}) {
	// TODO
}

func (l *Logger) Fatalln(v ...interface{}) {
	// TODO
}

// Panic logs an Err message, flushes the writer, and panics.
func (l *Logger) Panic(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Panicf(format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Panicln(v ...interface{}) {
	// TODO
}

// Print is equivalent to Log(Info, v...)
func (l *Logger) Print(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Printf(format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Println(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Err(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Errf(format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Errln(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Warn(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Warnf(format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Warnln(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Info(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Infof(format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Infoln(v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Debug(level uint8, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Debugln(level uint8, v ...interface{}) {
	// TODO
}

// SetLevel sets the debug level of this instance. After this call, this
// instance will log debug messages with level up to and including
// maxLevel. A negative maxLevel means that debug logging is turned off.
// A maxLevel >= 255 means all debug messages get logged.
func (l *Logger) SetLevel(maxLevel int) {
	// TODO
}
