package debuglogger

import (
	"github.com/Symantec/Dominator/lib/log"
)

type Logger struct {
	level int16
	log.Logger
}

// New will create a Logger from an existing log.Logger, adding methods for
// debug logs. Debug messages will be logged or ignored depending on their debug
// level. By default, the max debug level is -1, meaning all debug logs are
// dropped (ignored).
func New(logger log.Logger) *Logger {
	return &Logger{-1, logger}
}

// Upgrade will upgrade a log.Logger, adding methods for debug logs. If the
// provided logger is a log.DebugLogger, it is simply returned, otherwise the
// logger is wrapped in a new Logger.
func Upgrade(logger log.Logger) log.DebugLogger {
	if logger, ok := logger.(log.DebugLogger); ok {
		return logger
	}
	return New(logger)
}

// Debug will call the Print method if level is less than or equal to the max
// debug level for the Logger.
func (l *Logger) Debug(level uint8, v ...interface{}) {
	if l.level >= int16(level) {
		l.Print(v...)
	}
}

// Debugf will call the Printf method if level is less than or equal to the max
// debug level for the Logger.
func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	if l.level >= int16(level) {
		l.Printf(format, v...)
	}
}

// Debugln will call the Println method if level is less than or equal to the
// max debug level for the Logger.
func (l *Logger) Debugln(level uint8, v ...interface{}) {
	if l.level >= int16(level) {
		l.Println(v...)
	}
}

// GetLevel gets the current maximum debug level.
func (l *Logger) GetLevel() int16 {
	return l.level
}

// SetLevel sets the maximum debug level. A negative level will cause all debug
// messages to be dropped.
func (l *Logger) SetLevel(maxLevel int16) {
	l.level = maxLevel
}
