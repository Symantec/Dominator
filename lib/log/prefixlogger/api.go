package prefixlogger

import (
	"fmt"
	"github.com/Symantec/Dominator/lib/log"
)

type Logger struct {
	prefix string
	logger log.Logger
}

func New(prefix string, logger log.Logger) *Logger {
	return &Logger{prefix, logger}
}

func (l *Logger) Fatal(v ...interface{}) {
	l.logger.Fatal(l.prefix + fmt.Sprint(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal(l.prefix + fmt.Sprintf(format, v...))
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.logger.Fatal(l.prefix + fmt.Sprintln(v...))
}

func (l *Logger) Panic(v ...interface{}) {
	l.logger.Panic(l.prefix + fmt.Sprint(v...))
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	l.logger.Panic(l.prefix + fmt.Sprintf(format, v...))
}

func (l *Logger) Panicln(v ...interface{}) {
	l.logger.Panic(l.prefix + fmt.Sprintln(v...))
}

func (l *Logger) Print(v ...interface{}) {
	l.logger.Print(l.prefix + fmt.Sprint(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.logger.Print(l.prefix + fmt.Sprintf(format, v...))
}

func (l *Logger) Println(v ...interface{}) {
	l.logger.Print(l.prefix + fmt.Sprintln(v...))
}
