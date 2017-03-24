package debuglogger

import (
	"github.com/Symantec/Dominator/lib/log"
)

type Logger struct {
	level int16
	log.Logger
}

func New(logger log.Logger) *Logger {
	return &Logger{-1, logger}
}

func (l *Logger) Debug(level uint8, v ...interface{}) {
	if l.level >= int16(level) {
		l.Print(v)
	}
}

func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	if l.level >= int16(level) {
		l.Printf(format, v)
	}
}

func (l *Logger) Debugln(level uint8, v ...interface{}) {
	if l.level >= int16(level) {
		l.Println(v)
	}
}

func (l *Logger) SetLevel(maxLevel int16) {
	l.level = maxLevel
}
