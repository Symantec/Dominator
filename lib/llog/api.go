package llog

import (
	"io"
)

// Level indicates logging level. The higher the value, the less important.
// Any non-negative integer is always less important than a named constant.
type Level int

const (
	// Least important
	Info Level = -1 - iota
	Warn
	// Most important
	Err
)

// String returns the string representation of this level in all caps.
func (l *Level) String() string {
	// TODO
	return ""
}

// Logger instances log messages to a particular writer. A Logger instance is
// safe to use with multiple goroutines. Moreover, multiple instances logging
// to the same writer synchronise their writes.
type Logger struct {
	// TODO
}

// New creates a new logger, using --logLevel command-line flag to
// decide what messages get logged. If the --logLevel
// command-line flag is not present, NewDefault uses importance of Info.
// --logLevel command-line flag is of form level or subsystem:level. For
// example --logLevel Warn or --logLevel images:Info or --logLevel images:2.
// --logLevel may be used more than once on a command line. Log levels for
// subsystems e.g --logLevel images:Info always override global log level e.g
// --logLevel 2. In the presence of conflict such as --logLevel images:Info
// --logLevel images:Err, the last -logLevel, in this case images:Err, wins.
//
// writer is where the returned logger writes its messages. If the writer
// implementation has a no-arg Flush() method that returns an error,
// the returned logger calls it after logging a fatal or panic message
// just before terminating the process.
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
func (l *Logger) Log(level Level, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Logf(level Level, format string, v ...interface{}) {
	// TODO
}

// TODO
func (l *Logger) Logln(level Level, v ...interface{}) {
	// TODO
}

// Level returns the level of this log. That is, which messages it logs.
func (l *Logger) Level() Level {
	// TODO
	return Info
}
