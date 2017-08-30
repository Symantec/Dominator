package llog

// LoggerInterface is a simple interface for logging messages.
// LoggerInterface instances must be safe to use with multiple goroutines.
// Note that log.Logger already implements this interface.
type LoggerInterface interface {
	// calldepth is the call depth, usually 2. s is the message to log
	// without the timestamps.
	Output(calldepth int, s string) error
}

// Logger instances log messages to a particular LoggerInterface. A Logger
// instance is safe to use with multiple goroutines.
type Logger struct {
	// TODO
}

// New creates a new logger. New uses --logLevel command-line flag to
// decide what debug messages to log. If the --logLevel command-line flag
// is not present, the returned logger will not log any debug messages.
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
// writer is where the returned logger writes its messages.
func New(writer LoggerInterface, subSystemName string) *Logger {
	// TODO
	return nil
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
