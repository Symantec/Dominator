package testlogger

import (
	"fmt"
)

type Logger struct {
	logger TestLogger
}

type TestLogger interface {
	Error(v ...interface{})
	Errorf(format string, v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Log(v ...interface{})
	Logf(format string, v ...interface{})
}

func New(logger TestLogger) *Logger {
	return &Logger{logger}
}

func (l *Logger) Debug(level uint8, v ...interface{}) {
	l.logger.Log(sprint(v...))
}

func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	l.logger.Log(sprintf(format, v...))
}

func (l *Logger) Debugln(level uint8, v ...interface{}) {
	l.logger.Log(sprint(v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.logger.Fatal(sprint(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal(sprintf(format, v...))
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.logger.Fatal(sprint(v...))
}

func (l *Logger) Panic(v ...interface{}) {
	s := sprint(v...)
	l.logger.Fatal(s)
	panic(s)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	s := sprintf(format, v...)
	l.logger.Fatal(s)
	panic(s)
}

func (l *Logger) Panicln(v ...interface{}) {
	s := sprint(v...)
	l.logger.Fatal(s)
	panic(s)
}

func (l *Logger) Print(v ...interface{}) {
	l.logger.Log(sprint(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.logger.Log(sprintf(format, v...))
}

func (l *Logger) Println(v ...interface{}) {
	l.logger.Log(sprint(v...))
}

func strip(s string) string {
	length := len(s)
	if length < 1 {
		return ""
	}
	if s[length-1] == '\n' {
		return s[:length-1]
	}
	return s
}

func sprint(v ...interface{}) string {
	return strip(fmt.Sprint(v...))
}

func sprintf(format string, v ...interface{}) string {
	return strip(fmt.Sprintf(format, v...))
}
