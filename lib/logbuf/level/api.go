// Package level contains utilities for dealing with log levels
package level

import (
	"log"
)

// Level represents a log level
type Level string

const (
	None  Level = ""
	Trace Level = "TRACE"
	Debug Level = "DEBUG"
	Info  Level = "INFO"
	Warn  Level = "WARN"
	Err   Level = "ERR"
)

var (
	// Order is the order of log levels from least to greatest.
	Order = []Level{
		None,
		Trace,
		Debug,
		Info,
		Warn,
		Err,
	}
)

// Print prints a message with given level to l. Arguments are handled in the
// manner of fmt.Print
func Print(l *log.Logger, level Level, v ...interface{}) {
	l.Print(buildList(level, v)...)
}

// Printf prints a message with given level to l. Arguments are handled in the
// manner of fmt.Printf
func Printf(l *log.Logger, level Level, format string, v ...interface{}) {
	l.Printf(assign(level, format), v...)
}

// Println prints a message with given level to l. Arguments are handled in the
// manner of fmt.Println
func Println(l *log.Logger, level Level, v ...interface{}) {
	l.Println(buildList(level, v)...)
}

// Extract extracts the log level from a log line. The only requirement is
// that the log level be within the first pair of squre brackets. If no
// pair of square brackets exists, Extract returns None. Note
// that Extract(Assign(aLevel, aMessage)) -> aLevel.
func Extract(logLine string) Level {
	// TODO
	return None
}

func buildList(level Level, v []interface{}) []interface{} {
	// TODO
	return nil
}

func assign(level Level, format string) string {
	// TODO
	return ""
}
