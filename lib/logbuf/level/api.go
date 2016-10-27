// Package level contains utilities for dealing with log levels
package level

import (
	"fmt"
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

// Assign assigns a log level to a message.
// Assign(Warn, "Hi there!") -> "[WARN] hi there!"
func Assign(level Level, message string) string {
	if level != None {
		return fmt.Sprintf("[%s] %s", level, message)
	}
	return message
}

// Extract extracts the log level from a log line. The only requirement is
// that the log level be within the first pair of squre brackets. If no
// pair of square brackets exists, Extract returns None. Note
// that Extract(Assign(aLevel, aMessage)) -> aLevel.
func Extract(logLine string) Level {
	// TODO
	return None
}
