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
	l.logger.Log(fmt.Sprint(v...))
}

func (l *Logger) Debugf(level uint8, format string, v ...interface{}) {
	l.logger.Log(fmt.Sprintf(format, v...))
}

func (l *Logger) Debugln(level uint8, v ...interface{}) {
	l.logger.Log(fmt.Sprint(v...))
}

func (l *Logger) Fatal(v ...interface{}) {
	l.logger.Fatal(fmt.Sprint(v...))
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, v...))
}

func (l *Logger) Fatalln(v ...interface{}) {
	l.logger.Fatal(fmt.Sprint(v...))
}

func (l *Logger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.logger.Fatal(s)
	panic(s)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.logger.Fatal(s)
	panic(s)
}

func (l *Logger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	l.logger.Fatal(s)
	panic(s)
}

func (l *Logger) Print(v ...interface{}) {
	l.logger.Log(fmt.Sprint(v...))
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.logger.Log(fmt.Sprintf(format, v...))
}

func (l *Logger) Println(v ...interface{}) {
	l.logger.Log(fmt.Sprint(v...))
}
